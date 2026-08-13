package terminal

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/lmf112358/moyu-mahjiong-cli/internal/game"
)

func TestNormalTileLabels(t *testing.T) {
	want := map[game.Tile]string{0: "一萬", 8: "九萬", 9: "一筒", 18: "一索", 27: "東風", 31: "白板", 32: "發財", 33: "紅中"}
	seen := map[string]bool{}
	for tile := game.Tile(0); tile < 34; tile++ {
		got := normalTileLabel(tile)
		if seen[got] {
			t.Fatalf("duplicate glyph %q", got)
		}
		seen[got] = true
		if expected, ok := want[tile]; ok && got != expected {
			t.Errorf("tile %d=%q want %q", tile, got, expected)
		}
	}
}

func TestSettlementRenderIncludesScoring(t *testing.T) {
	var out bytes.Buffer
	c := &Controller{Out: &out, In: bufio.NewReader(strings.NewReader("\n")), Mode: StealthMode}
	c.ShowSettlement(game.Settlement{RoundWind: 27, HandNumber: 1, Wins: []game.WinDetail{{
		WinnerName: "甲", FromName: "乙", Tsumo: false, WinTile: 0, Structure: "七对子",
		Hand: []game.Tile{0, 0, 9, 9}, Yaku: []game.YakuItem{{Name: "立直", Han: 1}, {Name: "七对子", Han: 2}},
		Han: 3, Fu: 25, Limit: "", Gain: 3200, Dora: []game.Tile{8},
	}}, Changes: []game.ScoreChange{{Name: "甲", Before: 25000, Delta: 3200, After: 28200}, {Name: "乙", Before: 25000, Delta: -3200, After: 21800}}})
	s := out.String()
	for _, want := range []string{"和牌结算", "甲  荣和 ← 乙", "结构  七对子", "立直", "1翻", "七对子", "2翻", "合计  3翻 25符", "宝牌", "九→一", "+3200", "-3200"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q\n%s", want, s)
		}
	}
}

func TestNormalSettlementUsesLargeCards(t *testing.T) {
	var out bytes.Buffer
	c := &Controller{Out: &out, In: bufio.NewReader(strings.NewReader("\n")), Mode: NormalMode}
	c.ShowSettlement(game.Settlement{RoundWind: 27, HandNumber: 1, Wins: []game.WinDetail{{
		WinnerName: "甲", Tsumo: true, WinTile: 9, Structure: "面子手",
		Hand: []game.Tile{0, 1, 2, 9}, Melds: []game.Meld{{Kind: game.Chi, Tiles: []game.Tile{18, 19, 20}}},
		Yaku: []game.YakuItem{{Name: "门前清自摸和", Han: 1}}, Han: 1, Fu: 30, Gain: 1000,
	}}})
	s := out.String()
	for _, want := range []string{"東風1局", "和了牌 一筒", "│ 一  │", "│ 萬  │", "副露·吃", "│ 索  │"} {
		if !strings.Contains(s, want) {
			t.Errorf("normal settlement missing %q\n%s", want, s)
		}
	}
	if strings.Contains(s, "🀇") || strings.Contains(s, "🀐") {
		t.Fatalf("normal settlement fell back to tiny Unicode Mahjong glyphs:\n%s", s)
	}
}

func TestRenderShowsEveryRiverAndMeld(t *testing.T) {
	var out bytes.Buffer
	c := &Controller{Out: &out, Mode: NormalMode}
	d := game.Decision{View: game.View{
		Players: []game.Player{
			{Name: "甲", Hand: []game.Tile{0, 1}, River: []game.Tile{0}, Melds: []game.Meld{{Kind: game.Pon, Tiles: []game.Tile{27, 27, 27}}}},
			{Name: "乙", River: []game.Tile{9}, Melds: []game.Meld{{Kind: game.Chi, Tiles: []game.Tile{18, 19, 20}}}},
			{Name: "丙", River: []game.Tile{31}},
		},
		You: 0, Dealer: 0, RoundWind: 27, Dora: []game.Tile{8},
	}, Prompt: "请选择操作", Options: []game.Option{{Label: "打 一", Action: game.Action{Kind: game.ActDiscard, Tile: 0, Index: 0}}}}
	c.render(d)
	s := out.String()
	for _, expected := range []string{"[正常大牌]", "│ 一  │", "│ 筒  │", "│ 索  │", "│ 東  │", "│ 白  │", "副露·碰", "副露·吃", "[01]", "输入牌块下方数字出牌"} {
		if !strings.Contains(s, expected) {
			t.Errorf("render missing %q\n%s", expected, s)
		}
	}
	if strings.Count(s, "    河\n") != 3 {
		t.Fatalf("expected all three rivers, output=%s", s)
	}
	if strings.Contains(s, "打 一") || strings.Contains(s, "请选择操作") {
		t.Fatalf("duplicate discard options remain:\n%s", s)
	}
}

func TestStealthNotation(t *testing.T) {
	c := &Controller{Mode: StealthMode}
	if got := c.tiles([]game.Tile{0, 18, 9}); got != "一1①" {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(c.tiles([]game.Tile{0, 18, 9, 27}), "\033[") {
		t.Fatal("stealth tiles must not contain ANSI colors")
	}
	if ParseDisplayMode("2") != NormalMode || ParseDisplayMode("1") != StealthMode {
		t.Fatal("display selection parsing failed")
	}
}

func TestActivityAndLastDiscardCursor(t *testing.T) {
	var out bytes.Buffer
	c := &Controller{Out: &out, Mode: StealthMode}
	d := game.Decision{View: game.View{
		Players: []game.Player{{Name: "甲", Hand: []game.Tile{0}, River: []game.Tile{9}}, {Name: "乙"}},
		You:     0, Active: 1, LastFrom: 0, LastDiscard: 9, RoundWind: 27,
	}, Options: []game.Option{{Label: "跳过", Action: game.Action{Kind: game.ActPass}}}}
	c.render(d)
	s := out.String()
	if !strings.Contains(s, "当前 ▶ 乙") || !strings.Contains(s, "最近 甲 → ①") {
		t.Fatalf("missing table cursor: %s", s)
	}
}

func TestStealthRenderIsCompactAndColorless(t *testing.T) {
	var out bytes.Buffer
	c := &Controller{Out: &out, Mode: StealthMode}
	d := game.Decision{View: game.View{
		Players: []game.Player{
			{Name: "我", Hand: []game.Tile{0, 9, 18}, River: []game.Tile{27}},
			{Name: "AI", River: []game.Tile{31}, Melds: []game.Meld{{Kind: game.Pon, Tiles: []game.Tile{33, 33, 33}}}},
		}, You: 0, Active: 0, Dealer: 0, RoundWind: 27, HandNumber: 1, LastDiscard: 31, LastFrom: 1,
	}, Options: []game.Option{
		{Label: "拔北", Action: game.Action{Kind: game.ActKita, Tile: 30}},
		{Label: "打 一", Action: game.Action{Kind: game.ActDiscard, Tile: 0, Index: 0}},
		{Label: "打 ①", Action: game.Action{Kind: game.ActDiscard, Tile: 9, Index: 1}},
	}}
	c.render(d)
	s := out.String()
	withoutClear := strings.ReplaceAll(s, "\033[2J\033[H", "")
	if strings.Contains(withoutClear, "\033[") {
		t.Fatalf("stealth render contains ANSI styling: %q", withoutClear)
	}
	for _, want := range []string{"摸鱼雀  東1局", "当前 ▶ 我", "庄", "河│", ">白", "副 碰:中中中", "牌│一 │① │ 1 │", "键│01 │02 │03 │", "动作  [A] 拔北", "输入牌下方数字出牌"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q\n%s", want, s)
		}
	}
	if strings.Contains(s, "打 一") || strings.Contains(s, "\n出牌\n") || strings.Contains(s, "舍牌：") || strings.Contains(s, "牌桌动态") {
		t.Fatalf("obsolete repeated labels remain:\n%s", s)
	}
}

func TestStealthInputSeparatesHandKeysAndActions(t *testing.T) {
	d := game.Decision{Options: []game.Option{
		{Label: "拔北", Action: game.Action{Kind: game.ActKita, Tile: 30}},
		{Label: "打 一", Action: game.Action{Kind: game.ActDiscard, Tile: 0, Index: 0}},
		{Label: "打 ①", Action: game.Action{Kind: game.ActDiscard, Tile: 9, Index: 1}},
	}}
	if got, ok := slotInput(d, "1"); !ok || got.Kind != game.ActDiscard || got.Tile != 0 {
		t.Fatalf("hand key 1=%+v ok=%v", got, ok)
	}
	if got, ok := slotInput(d, "A"); !ok || got.Kind != game.ActKita {
		t.Fatalf("action A=%+v ok=%v", got, ok)
	}
}

func TestNormalCardsAreLargeAndAligned(t *testing.T) {
	got := normalCards([]game.Tile{0, 9, 18, 27, 31, 33}, 6, true, false)
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("normal card rows=%d\n%s", len(lines), got)
	}
	for i := 1; i < len(lines); i++ {
		if displayWidth(lines[i]) != displayWidth(lines[0]) {
			t.Fatalf("row %d width=%d want %d\n%s", i, displayWidth(lines[i]), displayWidth(lines[0]), got)
		}
	}
	for _, want := range []string{"│ 一  │", "│ 萬  │", "│ 東  │", "│ 風  │", "[06]"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q\n%s", want, got)
		}
	}
}

func TestStealthHandMarksDrawnTile(t *testing.T) {
	got := stealthHand([]game.Tile{0, 9, 18}, 1, true)
	if !strings.Contains(got, "摸入 02:①") {
		t.Fatalf("drawn tile not marked: %s", got)
	}
}

func TestStealthHandColumnsUseDisplayWidth(t *testing.T) {
	got := stealthHand([]game.Tile{0, 9, 10, 18, 27, 31}, 0, false)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("lines=%q", lines)
	}
	if displayWidth(lines[0]) != displayWidth(lines[1]) {
		t.Fatalf("visual widths differ: tiles=%d keys=%d\n%s", displayWidth(lines[0]), displayWidth(lines[1]), got)
	}
	if displayWidth("九") != 2 || displayWidth("①") != 2 || displayWidth("1") != 1 {
		t.Fatalf("unexpected terminal widths")
	}
}

func TestRiverGridUsesSixColumns(t *testing.T) {
	river := []game.Tile{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	got := stealthRiverGrid(river, true)
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("river rows=%d\n%s", len(lines), got)
	}
	if !strings.Contains(lines[2], ">④") {
		t.Fatalf("last discard not marked: %s", got)
	}
}

func TestReadyStatusExplainsOpenHand(t *testing.T) {
	hand := []game.Tile{0, 1, 2, 9, 10, 11, 27, 27, 27, 31, 31}
	options := make([]game.Option, len(hand))
	for i, tile := range hand {
		options[i] = game.Option{Action: game.Action{Kind: game.ActDiscard, Tile: tile, Index: i}}
	}
	d := game.Decision{View: game.View{Players: []game.Player{{Hand: hand, Melds: []game.Meld{{Kind: game.Kan, Tiles: []game.Tile{8, 8, 8, 8}}}}, {}, {}, {}}, You: 0}, Options: options}
	got := stealthReadyStatus(d)
	if !strings.Contains(got, "可听牌") || !strings.Contains(got, "不可立直：已有明副露") {
		t.Fatalf("status=%q", got)
	}
}
