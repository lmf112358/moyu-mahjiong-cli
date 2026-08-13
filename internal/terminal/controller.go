package terminal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/lmf112358/moyu-mahjiong-cli/internal/game"
)

type DisplayMode string

const (
	StealthMode DisplayMode = "stealth"
	NormalMode  DisplayMode = "normal"
)

type Controller struct {
	In    *bufio.Reader
	Out   io.Writer
	Mode  DisplayMode
	frame int
}

const ansiReset = "\033[0m"

func ParseDisplayMode(s string) DisplayMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "normal", "n", "2", "正常", "图形":
		return NormalMode
	default:
		return StealthMode
	}
}

func NewController() *Controller { return NewControllerWithMode(StealthMode) }
func NewControllerWithMode(mode DisplayMode) *Controller {
	return &Controller{In: bufio.NewReader(os.Stdin), Out: os.Stdout, Mode: mode}
}

func (c *Controller) Decide(d game.Decision) game.Action {
	c.render(d)
	for {
		fmt.Fprint(c.Out, "\n摸鱼雀 > ")
		line, err := c.In.ReadString('\n')
		if err != nil {
			return game.Action{Kind: game.ActQuit}
		}
		line = strings.TrimSpace(line)
		command := strings.ToLower(line)
		if command == "q" || command == "quit" {
			return game.Action{Kind: game.ActQuit}
		}
		if command == "h" || command == "help" {
			fmt.Fprintln(c.Out, "输入手牌下方数字出牌；特殊动作输入大写字母；a 自动出牌；q 退出。")
			continue
		}
		if command == "a" {
			for _, o := range d.Options {
				if o.Action.Kind == game.ActDiscard {
					return o.Action
				}
			}
		}
		if action, ok := slotInput(d, line); ok {
			return action
		}
		fmt.Fprint(c.Out, "请输入手牌数字或动作大写字母。")
	}
}

func slotInput(d game.Decision, input string) (game.Action, bool) {
	if len(input) == 1 {
		ch := input[0]
		if ch >= 'A' && ch <= 'Z' {
			special := specialOptions(d.Options)
			if i := int(ch - 'A'); i < len(special) {
				return special[i].Action, true
			}
		}
	}
	n, err := strconv.Atoi(input)
	if err != nil || n < 1 {
		return game.Action{}, false
	}
	hasDiscard := false
	for _, o := range d.Options {
		if o.Action.Kind == game.ActDiscard {
			hasDiscard = true
			if o.Action.Index == n-1 {
				return o.Action, true
			}
		}
	}
	if !hasDiscard && n <= len(d.Options) {
		return d.Options[n-1].Action, true
	}
	return game.Action{}, false
}

func specialOptions(options []game.Option) []game.Option {
	var result []game.Option
	for _, o := range options {
		if o.Action.Kind != game.ActDiscard {
			result = append(result, o)
		}
	}
	return result
}

func (c *Controller) render(d game.Decision) {
	if c.Mode == StealthMode {
		c.renderStealth(d)
		return
	}
	c.renderNormal(d)
}

func (c *Controller) renderNormal(d game.Decision) {
	v := d.View
	fmt.Fprint(c.Out, "\033[2J\033[H")
	fmt.Fprintf(c.Out, "摸鱼雀 [正常大牌]  %s%d局 · %d本场  余%d  宝:%s  供托:%d\n",
		v.RoundWind.String(), v.HandNumber, v.Honba, v.WallLeft, normalShortTiles(v.Dora), v.RiichiSticks)
	fmt.Fprintln(c.Out, normalActivity(v))
	fmt.Fprintln(c.Out, "════════════════════════════════════════════════════════════════════════════════════════════════════")
	for i, p := range v.Players {
		marker := "  "
		if i == v.Active {
			marker = "▶ "
		} else if i == v.You {
			marker = "● "
		}
		wind := game.Tile(27 + (i-v.Dealer+len(v.Players))%len(v.Players)).String()
		var state []string
		if i == v.Dealer {
			state = append(state, "庄")
		}
		if p.Riichi {
			state = append(state, "立直")
		}
		if p.Kita > 0 {
			state = append(state, fmt.Sprintf("北×%d", p.Kita))
		}
		fmt.Fprintf(c.Out, "%s%s %s %6d  %s\n", marker, wind, padDisplayRight(p.Name, 12), p.Score, strings.Join(state, " "))
		fmt.Fprintln(c.Out, "    河")
		fmt.Fprint(c.Out, normalCards(p.River, 6, false, i == v.LastFrom))
		for _, m := range p.Melds {
			fmt.Fprintf(c.Out, "    副露·%s\n", m.Kind)
			fmt.Fprint(c.Out, normalCards(m.Tiles, 6, false, false))
		}
	}
	fmt.Fprintln(c.Out, "════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Fprintln(c.Out, "你的手牌")
	fmt.Fprintln(c.Out, normalCards(v.Players[v.You].Hand, 14, true, false))
	if v.HasDrawn && v.DrawnIndex >= 0 && v.DrawnIndex < len(v.Players[v.You].Hand) {
		fmt.Fprintf(c.Out, "摸入 [%02d] %s\n", v.DrawnIndex+1, normalTileLabel(v.Players[v.You].Hand[v.DrawnIndex]))
	}
	if status := stealthReadyStatus(d); status != "" {
		fmt.Fprintln(c.Out, status)
	}
	fmt.Fprintln(c.Out)
	renderNormalOptions(c.Out, d)
}

func normalActivity(v game.View) string {
	active := "等待"
	if v.Active >= 0 && v.Active < len(v.Players) {
		active = v.Players[v.Active].Name
	}
	recent := "尚无舍牌"
	if v.LastDiscard < 34 && v.LastFrom >= 0 && v.LastFrom < len(v.Players) {
		recent = v.Players[v.LastFrom].Name + " → " + normalTileLabel(v.LastDiscard)
	}
	return "当前 ▶ " + active + "    最近 " + recent
}

func renderNormalOptions(w io.Writer, d game.Decision) {
	special := specialOptions(d.Options)
	if len(special) > 0 {
		fmt.Fprint(w, "动作  ")
		for i, o := range special {
			fmt.Fprintf(w, "[%c] %s   ", 'A'+i, normalOption(o))
		}
		fmt.Fprintln(w)
	}
	hasDiscard := false
	for _, o := range d.Options {
		if o.Action.Kind == game.ActDiscard {
			hasDiscard = true
			break
		}
	}
	if hasDiscard {
		fmt.Fprintln(w, "输入牌块下方数字出牌")
	} else if len(special) == 0 {
		fmt.Fprintln(w, d.Prompt)
	}
}

func normalOption(o game.Option) string {
	a := o.Action
	switch a.Kind {
	case game.ActRiichi:
		return "立直·" + normalTileLabel(a.Tile)
	case game.ActKita:
		return "拔北"
	case game.ActChi:
		return "吃·" + normalShortTiles(append(append([]game.Tile(nil), a.Tiles...), a.Tile))
	case game.ActPon:
		return "碰·" + normalTileLabel(a.Tile)
	case game.ActKan:
		return "杠·" + normalTileLabel(a.Tile)
	case game.ActTsumo:
		return "自摸"
	case game.ActRon:
		return "荣和"
	case game.ActPass:
		return "跳过"
	default:
		return o.Label
	}
}

func normalCards(ts []game.Tile, perRow int, indexed, highlightLast bool) string {
	if len(ts) == 0 {
		return "      -\n"
	}
	if perRow <= 0 {
		perRow = len(ts)
	}
	var out strings.Builder
	for start := 0; start < len(ts); start += perRow {
		end := start + perRow
		if end > len(ts) {
			end = len(ts)
		}
		for row := 0; row < 4; row++ {
			out.WriteString("      ")
			for i := start; i < end; i++ {
				top, bottom := normalTileFace(ts[i])
				highlight := highlightLast && i == len(ts)-1
				switch row {
				case 0:
					if highlight {
						out.WriteString("╔═════╗")
					} else {
						out.WriteString("┌─────┐")
					}
				case 1:
					if highlight {
						out.WriteString("║" + centerDisplay(top, 5) + "║")
					} else {
						out.WriteString("│" + centerDisplay(top, 5) + "│")
					}
				case 2:
					if highlight {
						out.WriteString("║" + centerDisplay(bottom, 5) + "║")
					} else {
						out.WriteString("│" + centerDisplay(bottom, 5) + "│")
					}
				case 3:
					if highlight {
						out.WriteString("╚═════╝")
					} else {
						out.WriteString("└─────┘")
					}
				}
			}
			out.WriteByte('\n')
		}
		if indexed {
			out.WriteString("      ")
			for i := start; i < end; i++ {
				out.WriteString(centerDisplay(fmt.Sprintf("[%02d]", i+1), 7))
			}
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func normalTileFace(t game.Tile) (string, string) {
	ranks := [...]string{"一", "二", "三", "四", "五", "六", "七", "八", "九"}
	if t < 27 {
		suit := [...]string{"萬", "筒", "索"}[t.Suit()]
		return ranks[t.Rank()-1], suit
	}
	tops := [...]string{"東", "南", "西", "北", "白", "發", "紅"}
	bottoms := [...]string{"風", "風", "風", "風", "板", "財", "中"}
	return tops[t-27], bottoms[t-27]
}

func normalTileLabel(t game.Tile) string {
	top, bottom := normalTileFace(t)
	return top + bottom
}

func normalShortTiles(ts []game.Tile) string {
	if len(ts) == 0 {
		return "-"
	}
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = normalTileLabel(t)
	}
	return strings.Join(parts, " ")
}

func (c *Controller) renderStealth(d game.Decision) {
	v := d.View
	fmt.Fprint(c.Out, "\033[2J\033[H")
	fmt.Fprintf(c.Out, "摸鱼雀  %s%d局 · %d本场  余%d  宝:%s  供托:%d\n",
		v.RoundWind.String(), v.HandNumber, v.Honba, v.WallLeft, rawTiles(v.Dora), v.RiichiSticks)
	fmt.Fprintln(c.Out, c.stealthActivity(v))
	fmt.Fprintln(c.Out, "────────────────────────────────────────────────────────────")
	for i, p := range v.Players {
		marker := "  "
		if i == v.Active {
			marker = "▶ "
		} else if i == v.You {
			marker = "● "
		}
		wind := game.Tile(27 + (i-v.Dealer+len(v.Players))%len(v.Players)).String()
		extra := ""
		if i == v.Dealer {
			extra += " 庄"
		}
		if p.Riichi {
			extra += " 立直"
		}
		if p.Kita > 0 {
			extra += fmt.Sprintf(" 北×%d", p.Kita)
		}
		fmt.Fprintf(c.Out, "%s%s %s %6d  %s\n",
			marker, wind, padDisplayRight(p.Name, 12), p.Score, strings.TrimSpace(extra))
		fmt.Fprint(c.Out, stealthRiverGrid(p.River, i == v.LastFrom))
		if len(p.Melds) > 0 {
			fmt.Fprintln(c.Out, "    副 "+rawMelds(p.Melds))
		}
	}
	fmt.Fprintln(c.Out, "────────────────────────────────────────────────────────────")
	fmt.Fprintln(c.Out, stealthHand(d.View.Players[d.View.You].Hand, v.DrawnIndex, v.HasDrawn))
	if status := stealthReadyStatus(d); status != "" {
		fmt.Fprintln(c.Out, status)
	}
	fmt.Fprintln(c.Out)
	c.renderStealthOptions(d)
}

func (c *Controller) renderStealthOptions(d game.Decision) {
	special := specialOptions(d.Options)
	if len(special) > 0 {
		fmt.Fprint(c.Out, "动作  ")
		for i, o := range special {
			fmt.Fprintf(c.Out, "[%c] %s   ", 'A'+i, rawOption(o))
		}
		fmt.Fprintln(c.Out)
	}
	hasDiscard := false
	for _, o := range d.Options {
		if o.Action.Kind == game.ActDiscard {
			hasDiscard = true
			break
		}
	}
	if hasDiscard {
		fmt.Fprintln(c.Out, "输入牌下方数字出牌")
	} else if len(special) == 0 {
		fmt.Fprintln(c.Out, d.Prompt)
	}
}

func (c *Controller) stealthActivity(v game.View) string {
	active := "等待"
	if v.Active >= 0 && v.Active < len(v.Players) {
		active = v.Players[v.Active].Name
	}
	recent := "尚无舍牌"
	if v.LastDiscard < 34 && v.LastFrom >= 0 && v.LastFrom < len(v.Players) {
		recent = v.Players[v.LastFrom].Name + " → " + v.LastDiscard.String()
	}
	return "当前 ▶ " + active + "    最近 " + recent
}

func stealthHand(ts []game.Tile, drawnIndex int, hasDrawn bool) string {
	var tiles, indices strings.Builder
	tiles.WriteString("牌│")
	indices.WriteString("键│")
	for i, t := range ts {
		tiles.WriteString(centerDisplay(t.String(), 3) + "│")
		indices.WriteString(centerDisplay(fmt.Sprintf("%02d", i+1), 3) + "│")
	}
	result := tiles.String() + "\n" + indices.String()
	if hasDrawn && drawnIndex >= 0 && drawnIndex < len(ts) {
		result += fmt.Sprintf("\n摸入 %02d:%s", drawnIndex+1, ts[drawnIndex].String())
	}
	return result
}

func centerDisplay(s string, width int) string {
	used := displayWidth(s)
	if used >= width {
		return s
	}
	left := (width - used) / 2
	right := width - used - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func padDisplayRight(s string, width int) string {
	if n := width - displayWidth(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if isCombining(r) {
			continue
		}
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

func isCombining(r rune) bool {
	return r >= 0x0300 && r <= 0x036f || r >= 0x1ab0 && r <= 0x1aff ||
		r >= 0x1dc0 && r <= 0x1dff || r >= 0x20d0 && r <= 0x20ff || r >= 0xfe20 && r <= 0xfe2f
}

func isWide(r rune) bool {
	return r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
		r >= 0x2e80 && r <= 0xa4cf && r != 0x303f ||
		r >= 0xac00 && r <= 0xd7a3 || r >= 0xf900 && r <= 0xfaff ||
		r >= 0xfe10 && r <= 0xfe19 || r >= 0xfe30 && r <= 0xfe6f ||
		r >= 0xff00 && r <= 0xff60 || r >= 0xffe0 && r <= 0xffe6 ||
		r >= 0x1f000 && r <= 0x1faff || r >= 0x20000 && r <= 0x3fffd)
}

func rawTiles(ts []game.Tile) string {
	var b strings.Builder
	for _, t := range ts {
		b.WriteString(t.String())
	}
	return b.String()
}

func rawTilesOrDash(ts []game.Tile) string {
	if len(ts) == 0 {
		return "-"
	}
	return rawTiles(ts)
}

func stealthRiverGrid(ts []game.Tile, highlightLast bool) string {
	if len(ts) == 0 {
		return "    河│ -\n"
	}
	const perRow = 6
	var b strings.Builder
	for start := 0; start < len(ts); start += perRow {
		if start == 0 {
			b.WriteString("    河│")
		} else {
			b.WriteString("      │")
		}
		end := start + perRow
		if end > len(ts) {
			end = len(ts)
		}
		for i := start; i < end; i++ {
			cell := ts[i].String()
			if highlightLast && i == len(ts)-1 {
				cell = ">" + cell
			}
			b.WriteString(centerDisplay(cell, 4) + "│")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func stealthReadyStatus(d game.Decision) string {
	p := d.View.Players[d.View.You]
	if p.Riichi {
		return "状态  已立直"
	}
	hasDiscard := false
	for _, o := range d.Options {
		if o.Action.Kind == game.ActDiscard {
			hasDiscard = true
			break
		}
	}
	if !hasDiscard {
		return ""
	}
	discards := game.TenpaiDiscards(p.Hand, len(p.Melds), len(d.View.Players))
	if len(discards) == 0 {
		return ""
	}
	var names []string
	seen := map[game.Tile]bool{}
	for _, i := range discards {
		if i >= 0 && i < len(p.Hand) && !seen[p.Hand[i]] {
			seen[p.Hand[i]] = true
			names = append(names, p.Hand[i].String())
		}
	}
	canRiichi := false
	for _, o := range d.Options {
		if o.Action.Kind == game.ActRiichi {
			canRiichi = true
			break
		}
	}
	if canRiichi {
		return "状态  可立直 · 切 " + strings.Join(names, "/")
	}
	reason := ""
	for _, m := range p.Melds {
		if m.Kind != game.Ankan {
			reason = "已有明副露"
			break
		}
	}
	if reason == "" && p.Score < 1000 {
		reason = "点数不足1000"
	}
	if reason == "" {
		reason = "规则限制"
	}
	return "状态  可听牌 · 切 " + strings.Join(names, "/") + " · 不可立直：" + reason
}

func rawMelds(ms []game.Meld) string {
	var parts []string
	for _, m := range ms {
		parts = append(parts, string(m.Kind)+":"+rawTiles(m.Tiles))
	}
	return strings.Join(parts, "  ")
}

func rawOption(o game.Option) string {
	a := o.Action
	switch a.Kind {
	case game.ActRiichi:
		return "立直·" + a.Tile.String()
	case game.ActKita:
		return "拔北"
	case game.ActChi:
		return "吃·" + rawTiles(append(append([]game.Tile(nil), a.Tiles...), a.Tile))
	case game.ActPon:
		return "碰·" + a.Tile.String()
	case game.ActKan:
		if len(a.Tiles) > 0 {
			return "杠·" + a.Tile.String()
		}
		return "暗杠·" + a.Tile.String()
	case game.ActTsumo:
		return "自摸"
	case game.ActRon:
		return "荣和"
	case game.ActPass:
		return "跳过"
	default:
		return o.Label
	}
}

func (c *Controller) modeName() string {
	if c.Mode == NormalMode {
		return "正常图形"
	}
	return "隐匿字符"
}

func (c *Controller) tile(t game.Tile) string {
	if t >= 34 {
		return t.String()
	}
	if c.Mode != NormalMode {
		return t.String()
	}
	var r rune
	switch {
	case t < 9:
		r = 0x1F007 + rune(t) // 万子
	case t < 18:
		r = 0x1F019 + rune(t-9) // 筒子
	case t < 27:
		r = 0x1F010 + rune(t-18) // 索子
	default:
		honors := [...]rune{0x1F000, 0x1F001, 0x1F002, 0x1F003, 0x1F006, 0x1F005, 0x1F004}
		r = honors[t-27]
	}
	return string(r)
}

func (c *Controller) tiles(ts []game.Tile) string {
	if len(ts) == 0 {
		return "（无）"
	}
	var b strings.Builder
	for _, t := range ts {
		b.WriteString(c.tile(t))
		if c.Mode == NormalMode {
			b.WriteByte(' ')
		}
	}
	return strings.TrimSpace(b.String())
}

func (c *Controller) indexedTiles(ts []game.Tile) string {
	if c.Mode == NormalMode {
		return c.largeTiles(ts, false, true)
	}
	var top, bottom strings.Builder
	for i, t := range ts {
		top.WriteString(fmt.Sprintf("〔%02d〕", i+1))
		bottom.WriteString("  " + c.tile(t) + "   ")
	}
	return strings.TrimRight(top.String(), " ") + "\n" + strings.TrimRight(bottom.String(), " ")
}

func (c *Controller) largeTiles(ts []game.Tile, highlightLast, indexed bool) string {
	if len(ts) == 0 {
		return "      （无）\n"
	}
	const perRow = 14
	var out strings.Builder
	for start := 0; start < len(ts); start += perRow {
		end := start + perRow
		if end > len(ts) {
			end = len(ts)
		}
		for row := 0; row < 3; row++ {
			out.WriteString("      ")
			for i := start; i < end; i++ {
				highlight := highlightLast && i == len(ts)-1
				switch row {
				case 0:
					if highlight {
						out.WriteString("╔═══╗")
					} else {
						out.WriteString("┌───┐")
					}
				case 1:
					if highlight {
						out.WriteString("║ " + c.tile(ts[i]) + " ║")
					} else {
						out.WriteString("│ " + c.tile(ts[i]) + " │")
					}
				case 2:
					if highlight {
						out.WriteString("╚═══╝")
					} else {
						out.WriteString("└───┘")
					}
				}
			}
			out.WriteByte('\n')
		}
		if indexed {
			out.WriteString("      ")
			for i := start; i < end; i++ {
				out.WriteString(fmt.Sprintf("〔%02d〕", i+1))
			}
			out.WriteByte('\n')
		}
	}
	return strings.TrimSuffix(out.String(), "\n")
}

func (c *Controller) stealthRiver(ts []game.Tile, highlightLast bool) string {
	if len(ts) == 0 {
		return "（无）"
	}
	var b strings.Builder
	for i, t := range ts {
		if highlightLast && i == len(ts)-1 {
			b.WriteString("\033[7m▸" + c.tile(t) + ansiReset)
		} else {
			b.WriteString(c.tile(t))
		}
	}
	return b.String()
}

func (c *Controller) tableActivity(v game.View) string {
	frames := [...]string{"◐", "◓", "◑", "◒"}
	active := "等待"
	if v.Active >= 0 && v.Active < len(v.Players) {
		active = v.Players[v.Active].Name
	}
	recent := "暂无舍牌"
	if v.LastDiscard < 34 && v.LastFrom >= 0 && v.LastFrom < len(v.Players) {
		recent = fmt.Sprintf("最近舍牌 %s ▸ %s", v.Players[v.LastFrom].Name, c.tile(v.LastDiscard))
	}
	return fmt.Sprintf("\033[1;35m%s 牌桌动态  ▶ %s 正在操作  │  %s\033[0m", frames[c.frame%len(frames)], active, recent)
}

func (c *Controller) option(o game.Option) string {
	a := o.Action
	switch a.Kind {
	case game.ActDiscard:
		return "打 " + c.tile(a.Tile)
	case game.ActRiichi:
		return "立直并打 " + c.tile(a.Tile)
	case game.ActKita:
		return "拔北 " + c.tile(a.Tile)
	case game.ActChi:
		return "吃 " + c.tiles(append(append([]game.Tile(nil), a.Tiles...), a.Tile))
	case game.ActPon:
		return "碰 " + c.tile(a.Tile)
	case game.ActKan:
		if len(a.Tiles) > 0 {
			return "杠 " + c.tiles(append(append([]game.Tile(nil), a.Tiles...), a.Tile))
		}
		return "暗杠 " + c.tile(a.Tile)
	default:
		return o.Label
	}
}

func (c *Controller) prompt(d game.Decision) string {
	for _, o := range d.Options {
		if o.Action.Kind == game.ActChi {
			return "可以吃 " + c.tile(o.Action.Tile)
		}
		if o.Action.Kind == game.ActPon || (o.Action.Kind == game.ActKan && len(o.Action.Tiles) > 0) {
			return "可以副露 " + c.tile(o.Action.Tile)
		}
	}
	return d.Prompt
}

func (c *Controller) message(s string) string {
	if c.Mode != NormalMode {
		return s
	}
	if !strings.Contains(s, " 打出 ") && !strings.Contains(s, "暗杠 ") {
		return s
	}
	for t := game.Tile(0); t < 34; t++ {
		if strings.HasSuffix(s, t.String()) {
			return strings.TrimSuffix(s, t.String()) + c.tile(t)
		}
	}
	return s
}

func PrintResult(w io.Writer, r game.MatchResult) {
	fmt.Fprintln(w, "\n=== 对局结果 ===")
	for i, p := range r.Players {
		fmt.Fprintf(w, "%d. %-12s %d点\n", i+1, p.Name, p.Score)
	}
}

func (c *Controller) ShowSettlement(s game.Settlement) {
	fmt.Fprint(c.Out, "\033[2J\033[H")
	fmt.Fprintf(c.Out, "和牌结算  %s%d局 · %d本场\n", c.settlementTile(s.RoundWind), s.HandNumber, s.Honba)
	fmt.Fprintln(c.Out, "════════════════════════════════════════════════════════════")
	for index, win := range s.Wins {
		method := "自摸"
		if !win.Tsumo {
			method = "荣和 ← " + win.FromName
		}
		fmt.Fprintf(c.Out, "%s  %s  和了牌 %s\n", win.WinnerName, method, c.settlementTile(win.WinTile))
		fmt.Fprintf(c.Out, "结构  %s\n", win.Structure)
		if c.Mode == NormalMode {
			fmt.Fprintln(c.Out, "手牌")
			fmt.Fprint(c.Out, normalCards(win.Hand, 14, false, false))
		} else {
			fmt.Fprintf(c.Out, "手牌  %s\n", c.settlementTiles(win.Hand))
		}
		if len(win.Melds) > 0 {
			if c.Mode == NormalMode {
				for _, m := range win.Melds {
					fmt.Fprintf(c.Out, "副露·%s\n", m.Kind)
					fmt.Fprint(c.Out, normalCards(m.Tiles, 6, false, false))
				}
			} else {
				fmt.Fprint(c.Out, "副露  ")
				for _, m := range win.Melds {
					fmt.Fprintf(c.Out, "[%s %s] ", m.Kind, c.settlementTiles(m.Tiles))
				}
				fmt.Fprintln(c.Out)
			}
		}
		fmt.Fprintln(c.Out, "役种")
		for _, y := range win.Yaku {
			if y.Yakuman > 0 {
				fmt.Fprintf(c.Out, "  %s 役满\n", padDisplayRight(y.Name, 18))
			} else {
				fmt.Fprintf(c.Out, "  %s %d翻\n", padDisplayRight(y.Name, 18), y.Han)
			}
		}
		if win.Yakuman > 0 {
			fmt.Fprintf(c.Out, "合计  %s  得点 +%d\n", win.Limit, win.Gain)
		} else {
			limit := win.Limit
			if limit != "" {
				limit = " · " + limit
			}
			fmt.Fprintf(c.Out, "合计  %d翻 %d符%s  得点 +%d\n", win.Han, win.Fu, limit, win.Gain)
		}
		fmt.Fprintf(c.Out, "宝牌  %s\n", indicatorSummary(c, win.Dora))
		if len(win.UraDora) > 0 {
			fmt.Fprintf(c.Out, "里宝  %s\n", indicatorSummary(c, win.UraDora))
		}
		if index+1 < len(s.Wins) {
			fmt.Fprintln(c.Out, "────────────────────────────────────────────────────────────")
		}
	}
	fmt.Fprintln(c.Out, "────────────────────────────────────────────────────────────")
	fmt.Fprintln(c.Out, "点数变化")
	for _, change := range s.Changes {
		fmt.Fprintf(c.Out, "  %s  %d  %s  → %d\n", padDisplayRight(change.Name, 12), change.Before, signed(change.Delta), change.After)
	}
	if s.RiichiSticks > 0 {
		fmt.Fprintf(c.Out, "供托  %d根（%d点）\n", s.RiichiSticks, s.RiichiSticks*1000)
	}
	fmt.Fprint(c.Out, "\n按 Enter 进入下一局…")
	if c.In != nil {
		_, _ = c.In.ReadString('\n')
	}
}

func (c *Controller) settlementTile(t game.Tile) string {
	if c.Mode == NormalMode {
		return normalTileLabel(t)
	}
	return t.String()
}

func (c *Controller) settlementTiles(ts []game.Tile) string {
	if c.Mode == NormalMode {
		return normalShortTiles(ts)
	}
	return rawTiles(ts)
}

func indicatorSummary(c *Controller, indicators []game.Tile) string {
	if len(indicators) == 0 {
		return "-"
	}
	var parts []string
	for _, ind := range indicators {
		parts = append(parts, c.settlementTile(ind)+"→"+c.settlementTile(game.DoraFrom(ind)))
	}
	return strings.Join(parts, "  ")
}

func signed(n int) string {
	if n > 0 {
		return fmt.Sprintf("+%d", n)
	}
	return fmt.Sprintf("%d", n)
}
