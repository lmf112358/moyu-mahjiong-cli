package game

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
)

type Config struct {
	Players    int
	Rounds     int // 1 = 东风战, 2 = 半庄
	StartScore int
	Seed       uint64
}

type MatchResult struct {
	Players []Player
	Log     []string
}

type Engine struct {
	Config                            Config
	Players                           []Player
	Controllers                       []Controller
	roundWind                         Tile
	handNumber, dealer, honba, sticks int
	wall, dead                        []Tile
	wallPos, rinshanPos, doraCount    int
	turn                              int
	lastDiscard                       Tile
	lastFrom                          int
	message                           string
	log                               []string
	firstTurn                         bool
	aborted                           bool
}

func NewEngine(cfg Config, names []string, cs []Controller) *Engine {
	if cfg.Players != 3 {
		cfg.Players = 4
	}
	if cfg.Rounds < 1 || cfg.Rounds > 2 {
		cfg.Rounds = 1
	}
	if cfg.StartScore == 0 {
		if cfg.Players == 3 {
			cfg.StartScore = 35000
		} else {
			cfg.StartScore = 25000
		}
	}
	e := &Engine{Config: cfg, Controllers: cs, roundWind: 27, lastDiscard: NoTile}
	for i := 0; i < cfg.Players; i++ {
		name := fmt.Sprintf("玩家%d", i+1)
		if i < len(names) && names[i] != "" {
			name = names[i]
		}
		e.Players = append(e.Players, Player{Name: name, Score: cfg.StartScore})
	}
	return e
}

func (e *Engine) Run() MatchResult {
	maxHands := e.Config.Rounds * e.Config.Players
	for e.handNumber < maxHands {
		dealerContinues, _ := e.playHand()
		if e.aborted {
			break
		}
		if dealerContinues {
			e.honba++
		} else {
			e.dealer = (e.dealer + 1) % e.Config.Players
			e.handNumber++
			e.honba = 0
		}
		if e.handNumber >= e.Config.Players {
			e.roundWind = 28
		}
		broke := false
		for _, p := range e.Players {
			if p.Score < 0 {
				broke = true
			}
		}
		if broke {
			break
		}
	}
	if e.sticks > 0 {
		top := 0
		for i := range e.Players {
			if e.Players[i].Score > e.Players[top].Score {
				top = i
			}
		}
		e.Players[top].Score += e.sticks * 1000
		e.sticks = 0
	}
	e.say("对局结束")
	return MatchResult{Players: e.Players, Log: e.log}
}

func (e *Engine) playHand() (dealerWon bool, exhaustive bool) {
	e.setupHand()
	e.say(fmt.Sprintf("%s%d局 %d本场，庄家 %s", e.roundWind.String(), e.handNumber%e.Config.Players+1, e.honba, e.Players[e.dealer].Name))
	e.turn = e.dealer
	e.firstTurn = true
	for {
		if e.liveLeft() <= 0 {
			return e.exhaustiveDraw(), true
		}
		drawn := e.drawLive(e.turn)
		if drawn == NoTile {
			return e.exhaustiveDraw(), true
		}
		var discard Tile
		for {
			outcome, chosen := e.drawDecision(e.turn, drawn)
			if outcome == ActQuit {
				e.aborted = true
				e.say(e.Players[e.turn].Name + " 离开了牌局")
				return false, false
			}
			if outcome == ActTsumo {
				e.win([]int{e.turn}, e.turn, true, drawn, false)
				return e.turn == e.dealer, false
			}
			if outcome == ActKita || outcome == ActKan {
				if len(e.Players[e.turn].Hand) == 0 {
					return false, false
				}
				drawn = e.Players[e.turn].Hand[len(e.Players[e.turn].Hand)-1]
				continue
			}
			discard = chosen
			break
		}
		e.discard(e.turn, discard)
		from, tile := e.turn, discard
		for {
			winners, caller, kind, used := e.reactions(from, tile)
			if len(winners) > 0 {
				e.win(winners, from, false, tile, false)
				return slices.Contains(winners, e.dealer), false
			}
			if caller < 0 {
				e.turn = (from + 1) % e.Config.Players
				break
			}
			e.call(caller, from, kind, tile, used)
			e.turn = caller
			out, nextDiscard := e.afterCallDecision(caller)
			if out == ActQuit {
				e.aborted = true
				return false, false
			}
			e.discard(caller, nextDiscard)
			from, tile = caller, nextDiscard
		}
		e.firstTurn = false
	}
}

func (e *Engine) setupHand() {
	wall := FullWall(e.Config.Players)
	if e.Config.Seed != 0 {
		r := rand.New(rand.NewPCG(e.Config.Seed+uint64(e.handNumber+e.honba), e.Config.Seed^0x9e3779b97f4a7c15))
		r.Shuffle(len(wall), func(i, j int) { wall[i], wall[j] = wall[j], wall[i] })
	} else {
		rand.Shuffle(len(wall), func(i, j int) { wall[i], wall[j] = wall[j], wall[i] })
	}
	e.dead = append([]Tile(nil), wall[len(wall)-14:]...)
	e.wall = wall[:len(wall)-14]
	e.wallPos = 0
	e.rinshanPos = 0
	e.doraCount = 1
	e.lastDiscard = NoTile
	for i := range e.Players {
		e.Players[i].Hand = nil
		e.Players[i].Melds = nil
		e.Players[i].River = nil
		e.Players[i].Riichi = false
		e.Players[i].Ippatsu = false
		e.Players[i].Furiten = false
		e.Players[i].Kita = 0
	}
	for n := 0; n < 13; n++ {
		for i := 0; i < e.Config.Players; i++ {
			e.Players[i].Hand = append(e.Players[i].Hand, e.wall[e.wallPos])
			e.wallPos++
		}
	}
	for i := range e.Players {
		SortTiles(e.Players[i].Hand)
	}
}

func (e *Engine) drawLive(i int) Tile {
	if e.wallPos >= len(e.wall) {
		return NoTile
	}
	t := e.wall[e.wallPos]
	e.wallPos++
	e.Players[i].Hand = append(e.Players[i].Hand, t)
	SortTiles(e.Players[i].Hand)
	return t
}
func (e *Engine) drawRinshan(i int) Tile {
	if e.rinshanPos >= 4 {
		return NoTile
	}
	t := e.dead[e.rinshanPos]
	e.rinshanPos++
	e.Players[i].Hand = append(e.Players[i].Hand, t)
	SortTiles(e.Players[i].Hand)
	return t
}
func (e *Engine) liveLeft() int { return len(e.wall) - e.wallPos - e.rinshanPos }
func (e *Engine) doraIndicators() []Tile {
	out := []Tile{}
	for i := 0; i < e.doraCount && 4+i*2 < len(e.dead); i++ {
		out = append(out, e.dead[4+i*2])
	}
	return out
}
func (e *Engine) uraIndicators() []Tile {
	out := []Tile{}
	for i := 0; i < e.doraCount && 5+i*2 < len(e.dead); i++ {
		out = append(out, e.dead[5+i*2])
	}
	return out
}

func (e *Engine) drawDecision(i int, drawn Tile) (ActionKind, Tile) {
	p := &e.Players[i]
	opts := []Option{}
	ctx := e.context(i, i, true, drawn)
	if _, ok := EvaluateWin(*p, ctx); ok {
		opts = append(opts, Option{"自摸", Action{Kind: ActTsumo, Tile: drawn}})
	}
	if !p.Riichi {
		for _, t := range uniqueTiles(p.Hand) {
			if countTile(p.Hand, t) == 4 {
				opts = append(opts, Option{"暗杠 " + t.String(), Action{Kind: ActKan, Tile: t}})
			}
		}
		if e.Config.Players == 3 && countTile(p.Hand, 30) > 0 {
			opts = append(opts, Option{"拔北", Action{Kind: ActKita, Tile: 30}})
		}
		if isClosed(*p) && p.Score >= 1000 {
			for _, idx := range TenpaiDiscards(p.Hand, len(p.Melds), e.Config.Players) {
				opts = append(opts, Option{"立直并打 " + p.Hand[idx].String(), Action{Kind: ActRiichi, Tile: p.Hand[idx], Index: idx}})
			}
		}
	}
	if p.Riichi {
		idx := lastIndexOf(p.Hand, drawn)
		opts = append(opts, Option{"摸切 " + drawn.String(), Action{Kind: ActDiscard, Tile: drawn, Index: idx}})
	} else {
		for idx, t := range p.Hand {
			opts = append(opts, Option{fmt.Sprintf("打 %s", t.String()), Action{Kind: ActDiscard, Tile: t, Index: idx}})
		}
	}
	a := e.decideDraw(i, "请选择操作", opts, lastIndexOf(p.Hand, drawn))
	switch a.Kind {
	case ActTsumo:
		return ActTsumo, drawn
	case ActQuit:
		return ActQuit, NoTile
	case ActKita:
		e.doKita(i)
		return ActKita, NoTile
	case ActKan:
		e.doAnkan(i, a.Tile)
		return ActKan, NoTile
	case ActRiichi:
		p.Score -= 1000
		e.sticks++
		p.Riichi = true
		p.Ippatsu = true
		e.say(p.Name + " 立直！")
		return ActDiscard, a.Tile
	default:
		return ActDiscard, a.Tile
	}
}

func (e *Engine) afterCallDecision(i int) (ActionKind, Tile) {
	p := &e.Players[i]
	opts := []Option{}
	for idx, t := range p.Hand {
		opts = append(opts, Option{"打 " + t.String(), Action{Kind: ActDiscard, Tile: t, Index: idx}})
	}
	a := e.decide(i, "副露后请选择打牌", opts)
	return a.Kind, a.Tile
}

func (e *Engine) reactions(from int, t Tile) ([]int, int, MeldKind, []Tile) {
	winners := []int{}
	for d := 1; d < e.Config.Players; d++ {
		i := (from + d) % e.Config.Players
		p := &e.Players[i]
		h := append(append([]Tile(nil), p.Hand...), t)
		p.Furiten = e.isFuriten(i)
		if !p.Furiten && IsComplete(h, len(p.Melds)) {
			copyP := *p
			copyP.Hand = h
			if _, ok := EvaluateWin(copyP, e.context(i, from, false, t)); ok {
				a := e.decide(i, "可荣和", []Option{{"荣和", Action{Kind: ActRon, Tile: t}}, {"跳过", Action{Kind: ActPass}}})
				if a.Kind == ActRon {
					winners = append(winners, i)
				} else {
					p.Furiten = true
				}
			}
		}
	}
	if len(winners) > 0 {
		return winners, -1, "", nil
	}
	// Pon/kan priority in turn order.
	for d := 1; d < e.Config.Players; d++ {
		i := (from + d) % e.Config.Players
		p := &e.Players[i]
		if p.Riichi {
			continue
		}
		opts := []Option{}
		if countTile(p.Hand, t) >= 3 {
			opts = append(opts, Option{"大明杠", Action{Kind: ActKan, Tile: t, Tiles: []Tile{t, t, t}}})
		}
		if countTile(p.Hand, t) >= 2 {
			opts = append(opts, Option{"碰", Action{Kind: ActPon, Tile: t, Tiles: []Tile{t, t}}})
		}
		if len(opts) > 0 {
			opts = append(opts, Option{"跳过", Action{Kind: ActPass}})
			a := e.decide(i, "可副露 "+t.String(), opts)
			if a.Kind == ActKan {
				return nil, i, Kan, a.Tiles
			}
			if a.Kind == ActPon {
				return nil, i, Pon, a.Tiles
			}
		}
	}
	// Only the next player may chi.
	i := (from + 1) % e.Config.Players
	if !e.Players[i].Riichi && !t.IsHonor() {
		combos := chiCombos(e.Players[i].Hand, t)
		if len(combos) > 0 {
			opts := []Option{}
			for _, c := range combos {
				opts = append(opts, Option{"吃 " + TilesString(append(c, t)), Action{Kind: ActChi, Tile: t, Tiles: c}})
			}
			opts = append(opts, Option{"跳过", Action{Kind: ActPass}})
			a := e.decide(i, "可吃 "+t.String(), opts)
			if a.Kind == ActChi {
				return nil, i, Chi, a.Tiles
			}
		}
	}
	return nil, -1, "", nil
}

func (e *Engine) discard(i int, t Tile) {
	p := &e.Players[i]
	var ok bool
	p.Hand, ok = removeTiles(p.Hand, t)
	if !ok && len(p.Hand) > 0 {
		t = p.Hand[len(p.Hand)-1]
		p.Hand = p.Hand[:len(p.Hand)-1]
	}
	p.River = append(p.River, t)
	e.lastDiscard = t
	e.lastFrom = i
	for j := range e.Players {
		if j != i {
			e.Players[j].Ippatsu = false
		}
	}
	e.say(fmt.Sprintf("%s 打出 %s", p.Name, t.String()))
}
func (e *Engine) call(i, from int, k MeldKind, t Tile, used []Tile) {
	p := &e.Players[i]
	var ok bool
	p.Hand, ok = removeTiles(p.Hand, used...)
	if !ok {
		return
	}
	tiles := append(append([]Tile(nil), used...), t)
	SortTiles(tiles)
	p.Melds = append(p.Melds, Meld{Kind: k, Tiles: tiles, From: from})
	if len(e.Players[from].River) > 0 {
		e.Players[from].River = e.Players[from].River[:len(e.Players[from].River)-1]
	}
	for j := range e.Players {
		e.Players[j].Ippatsu = false
	}
	e.say(fmt.Sprintf("%s %s！", p.Name, k))
	if k == Kan {
		e.doraCount++
		e.drawRinshan(i)
	}
}
func (e *Engine) doAnkan(i int, t Tile) {
	p := &e.Players[i]
	p.Hand, _ = removeTiles(p.Hand, t, t, t, t)
	p.Melds = append(p.Melds, Meld{Kind: Ankan, Tiles: []Tile{t, t, t, t}, From: i})
	e.doraCount++
	for j := range e.Players {
		e.Players[j].Ippatsu = false
	}
	e.drawRinshan(i)
	e.say(p.Name + " 暗杠 " + t.String())
}
func (e *Engine) doKita(i int) {
	p := &e.Players[i]
	p.Hand, _ = removeTiles(p.Hand, 30)
	p.Kita++
	e.drawLive(i)
	e.say(p.Name + " 拔北")
}

func (e *Engine) win(winners []int, from int, tsumo bool, t Tile, chankan bool) {
	before := make([]int, len(e.Players))
	for i := range e.Players {
		before[i] = e.Players[i].Score
	}
	sticksBefore := e.sticks
	settlement := Settlement{RoundWind: e.roundWind, HandNumber: e.handNumber%e.Config.Players + 1, Honba: e.honba, RiichiSticks: sticksBefore}
	for winnerIndex, w := range winners {
		p := &e.Players[w]
		if !tsumo {
			p.Hand = append(p.Hand, t)
			SortTiles(p.Hand)
		}
		ctx := e.context(w, from, tsumo, t)
		ctx.Chankan = chankan
		res, _ := EvaluateWin(*p, ctx)
		gain := 0
		if tsumo {
			for i := range e.Players {
				if i == w {
					continue
				}
				pay := res.TsumoChild
				if i == e.dealer {
					pay = res.TsumoDealer
				}
				pay += e.honba * 100
				e.Players[i].Score -= pay
				gain += pay
			}
		} else {
			pay := res.Ron + e.honba*300
			e.Players[from].Score -= pay
			gain = pay
		}
		if winnerIndex == 0 {
			gain += e.sticks * 1000
		}
		p.Score += gain
		detail := WinDetail{
			Winner: w, WinnerName: p.Name, From: from, Tsumo: tsumo, WinTile: t, Structure: winStructure(res),
			Hand: append([]Tile(nil), p.Hand...), Melds: cloneMelds(p.Melds),
			Yaku: append([]YakuItem(nil), res.YakuItems...), Han: res.Han, Fu: res.Fu,
			Yakuman: res.Yakuman, Limit: LimitName(res), Gain: gain, Kita: p.Kita,
			Dora: append([]Tile(nil), ctx.DoraIndicators...),
		}
		if !tsumo {
			detail.FromName = e.Players[from].Name
		}
		if p.Riichi {
			detail.UraDora = append([]Tile(nil), ctx.UraIndicators...)
		}
		settlement.Wins = append(settlement.Wins, detail)
		e.say(fmt.Sprintf("%s %s！%d翻%d符 +%d（%s）", p.Name, map[bool]string{true: "自摸", false: "荣和"}[tsumo], res.Han, res.Fu, gain, join(res.Yaku)))
		if !tsumo {
			p.Hand, _ = removeTiles(p.Hand, t)
		}
	}
	e.sticks = 0
	for i, p := range e.Players {
		settlement.Changes = append(settlement.Changes, ScoreChange{Player: i, Name: p.Name, Before: before[i], Delta: p.Score - before[i], After: p.Score})
	}
	e.showSettlement(settlement)
}

func winStructure(r ScoreResult) string {
	for _, y := range r.YakuItems {
		if y.Name == "国士无双" {
			return "国士无双"
		}
		if y.Name == "七对子" {
			return "七对子"
		}
	}
	return "四面子一雀头"
}

func cloneMelds(ms []Meld) []Meld {
	out := make([]Meld, len(ms))
	for i, m := range ms {
		out[i] = m
		out[i].Tiles = append([]Tile(nil), m.Tiles...)
	}
	return out
}

func (e *Engine) showSettlement(s Settlement) {
	var wg sync.WaitGroup
	for _, controller := range e.Controllers {
		viewer, ok := controller.(SettlementViewer)
		if !ok {
			continue
		}
		wg.Add(1)
		go func() { defer wg.Done(); viewer.ShowSettlement(s) }()
	}
	wg.Wait()
}

func (e *Engine) exhaustiveDraw() bool {
	tenpai := []int{}
	for i, p := range e.Players {
		if len(Waits(p.Hand, len(p.Melds), e.Config.Players)) > 0 {
			tenpai = append(tenpai, i)
		}
	}
	if len(tenpai) > 0 && len(tenpai) < e.Config.Players {
		pay := 3000 / (e.Config.Players - len(tenpai))
		gain := 3000 / len(tenpai)
		for i := range e.Players {
			if slices.Contains(tenpai, i) {
				e.Players[i].Score += gain
			} else {
				e.Players[i].Score -= pay
			}
		}
	}
	names := []string{}
	for _, i := range tenpai {
		names = append(names, e.Players[i].Name)
	}
	e.say("荒牌流局，听牌：" + join(names))
	return slices.Contains(tenpai, e.dealer)
}

func (e *Engine) context(w, from int, tsumo bool, t Tile) WinContext {
	return WinContext{Winner: w, From: from, WinTile: t, Tsumo: tsumo, Riichi: e.Players[w].Riichi, Ippatsu: e.Players[w].Ippatsu, Dealer: w == e.dealer, SeatWind: Tile(27 + (w-e.dealer+e.Config.Players)%e.Config.Players), RoundWind: e.roundWind, DoraIndicators: e.doraIndicators(), UraIndicators: e.uraIndicators(), Haitei: tsumo && e.liveLeft() == 0, Houtei: !tsumo && e.liveLeft() == 0, Players: e.Config.Players}
}
func (e *Engine) isFuriten(i int) bool {
	p := e.Players[i]
	waits := Waits(p.Hand, len(p.Melds), e.Config.Players)
	for _, t := range p.River {
		if contains(waits, t) {
			return true
		}
	}
	return false
}
func (e *Engine) decide(i int, prompt string, opts []Option) Action {
	v := e.viewFor(i)
	v.Active = i
	d := Decision{View: v, Prompt: prompt, Options: opts}
	a := e.Controllers[i].Decide(d)
	if a.Kind == ActQuit {
		return a
	}
	for _, o := range opts {
		if sameAction(a, o.Action) {
			return o.Action
		}
	}
	return opts[len(opts)-1].Action
}

func (e *Engine) decideDraw(i int, prompt string, opts []Option, drawnIndex int) Action {
	v := e.viewFor(i)
	v.Active = i
	v.DrawnIndex = drawnIndex
	v.HasDrawn = true
	d := Decision{View: v, Prompt: prompt, Options: opts}
	a := e.Controllers[i].Decide(d)
	if a.Kind == ActQuit {
		return a
	}
	for _, o := range opts {
		if sameAction(a, o.Action) {
			return o.Action
		}
	}
	return opts[len(opts)-1].Action
}
func (e *Engine) viewFor(i int) View {
	ps := make([]Player, len(e.Players))
	for j, p := range e.Players {
		ps[j] = p
		if j != i {
			ps[j].Hand = make([]Tile, len(p.Hand))
		}
	}
	return View{Players: ps, You: i, Turn: e.turn, Dealer: e.dealer, RoundWind: e.roundWind, HandNumber: e.handNumber%e.Config.Players + 1, Honba: e.honba, RiichiSticks: e.sticks, WallLeft: e.liveLeft(), Dora: e.doraIndicators(), Message: e.message, LastDiscard: e.lastDiscard, LastFrom: e.lastFrom}
}
func (e *Engine) say(s string) { e.message = s; e.log = append(e.log, s) }

func isClosed(p Player) bool {
	for _, m := range p.Melds {
		if m.Kind != Ankan {
			return false
		}
	}
	return true
}
func lastIndexOf(ts []Tile, t Tile) int {
	for i := len(ts) - 1; i >= 0; i-- {
		if ts[i] == t {
			return i
		}
	}
	return 0
}
func sameAction(a, b Action) bool {
	if a.Kind != b.Kind {
		return false
	}
	if b.Kind == ActDiscard || b.Kind == ActRiichi || b.Kind == ActKan || b.Kind == ActKita {
		return a.Tile == b.Tile
	}
	if b.Kind == ActChi {
		return slices.Equal(a.Tiles, b.Tiles)
	}
	return true
}
func chiCombos(hand []Tile, t Tile) [][]Tile {
	var out [][]Tile
	if t.IsHonor() {
		return out
	}
	r := t.Rank()
	cands := [][]int{{r - 2, r - 1}, {r - 1, r + 1}, {r + 1, r + 2}}
	base := int(t) - r + 1
	for _, c := range cands {
		if c[0] >= 1 && c[1] <= 9 {
			a, b := Tile(base+c[0]-1), Tile(base+c[1]-1)
			if countTile(hand, a) > 0 && countTile(hand, b) > 0 {
				out = append(out, []Tile{a, b})
			}
		}
	}
	return out
}
func join(xs []string) string {
	if len(xs) == 0 {
		return "无"
	}
	s := xs[0]
	for _, x := range xs[1:] {
		s += "、" + x
	}
	return s
}
