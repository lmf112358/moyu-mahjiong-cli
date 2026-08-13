package game

import "testing"

// 红五: 34=红五万 35=红五筒 36=红五索 (对应 Aka5Man/Aka5Pin/Aka5Sou)

func TestAkaTileMethods(t *testing.T) {
	cases := []struct {
		t    Tile
		base Tile
		aka  bool
		suit int
		rank int
	}{
		{Aka5Man, 4, true, 0, 5},
		{Aka5Pin, 13, true, 1, 5},
		{Aka5Sou, 22, true, 2, 5},
		{4, 4, false, 0, 5},
		{13, 13, false, 1, 5},
		{0, 0, false, 0, 1},
	}
	for _, c := range cases {
		if c.t.Base() != c.base {
			t.Errorf("%d.Base()=%d want %d", c.t, c.t.Base(), c.base)
		}
		if c.t.IsAka() != c.aka {
			t.Errorf("%d.IsAka()=%v want %v", c.t, c.t.IsAka(), c.aka)
		}
		if c.t.Suit() != c.suit || c.t.Rank() != c.rank {
			t.Errorf("%d Suit=%d Rank=%d want %d/%d", c.t, c.t.Suit(), c.t.Rank(), c.suit, c.rank)
		}
	}
}

func TestAkaFullWall(t *testing.T) {
	w4 := FullWall(4)
	if len(w4) != 136 {
		t.Fatalf("四人牌墙长度 %d want 136", len(w4))
	}
	aka, man, pin, sou := 0, 0, 0, 0
	for _, x := range w4 {
		if x.IsAka() {
			aka++
		}
		switch x {
		case Aka5Man:
			man++
		case Aka5Pin:
			pin++
		case Aka5Sou:
			sou++
		}
	}
	if aka != 3 || man != 1 || pin != 1 || sou != 1 {
		t.Fatalf("四人 aka 总%d (万%d 筒%d 索%d) want 3/1/1/1", aka, man, pin, sou)
	}

	w3 := FullWall(3)
	if len(w3) != 108 {
		t.Fatalf("三人牌墙长度 %d want 108", len(w3))
	}
	aka3, man3 := 0, 0
	for _, x := range w3 {
		if x.IsAka() {
			aka3++
		}
		if x == Aka5Man {
			man3++
		}
	}
	if aka3 != 2 {
		t.Fatalf("三人 aka %d want 2(筒+索)", aka3)
	}
	if man3 != 0 {
		t.Fatalf("三人麻将不应含红五万，got %d", man3)
	}
}

func TestAkaSortTiles(t *testing.T) {
	xs := tiles(34, 4)
	SortTiles(xs)
	if xs[0] != 4 || xs[1] != Aka5Man {
		t.Fatalf("排序后 %v，普通五应在前、红五在后", xs)
	}
}

func TestAkaCountsAndComplete(t *testing.T) {
	c := Counts(tiles(4, 34))
	if c[4] != 2 {
		t.Fatalf("Counts 五万(含红)=%d want 2", c[4])
	}
	// 123万 234筒 234索 东东东 (五万+红五万) = 和牌
	hand := tiles(0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 27, 27, 4, 34)
	if !IsComplete(hand, 0) {
		t.Fatalf("含红五万的对子手应判和牌")
	}
}

func TestAkaDoraScoring(t *testing.T) {
	// 立直和牌 + 手牌含红五万 → 立直1 + 赤宝牌1
	hand := tiles(0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 27, 27, 4, 34)
	p := Player{Hand: hand}
	ctx := WinContext{Riichi: true, SeatWind: 27, RoundWind: 27, Players: 4}
	res, ok := EvaluateWin(p, ctx)
	if !ok {
		t.Fatalf("应判和牌")
	}
	if res.Han < 2 {
		t.Fatalf("Han=%d，立直+赤宝牌应>=2", res.Han)
	}
	found := false
	for _, y := range res.YakuItems {
		if y.Name == "赤宝牌" && y.Han == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("缺赤宝牌役种: %v", res.YakuItems)
	}
}

func TestAkaDoraStacksWithIndicators(t *testing.T) {
	// 表宝牌=五万(指示四万3) + 手牌红五万 → 表宝牌1 + 赤宝牌1
	hand := tiles(0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 27, 27, 4, 34)
	p := Player{Hand: hand}
	ctx := WinContext{Riichi: true, DoraIndicators: []Tile{3}, SeatWind: 27, RoundWind: 27, Players: 4}
	res, ok := EvaluateWin(p, ctx)
	if !ok {
		t.Fatalf("应判和牌")
	}
	dora, aka := 0, 0
	for _, y := range res.YakuItems {
		if y.Name == "宝牌" {
			dora = y.Han
		}
		if y.Name == "赤宝牌" {
			aka = y.Han
		}
	}
	if dora != 2 {
		t.Fatalf("表宝牌翻=%d want 2(普通五万+红五万都算表宝牌五万)", dora)
	}
	if aka != 1 {
		t.Fatalf("赤宝牌翻=%d want 1", aka)
	}
}

func TestAkaCountTileNorm(t *testing.T) {
	hand := tiles(4, 34, 13)
	if got := countTileNorm(hand, 4); got != 2 {
		t.Fatalf("countTileNorm(五万)=%d want 2(含红)", got)
	}
	if got := countTileNorm(hand, 13); got != 1 {
		t.Fatalf("countTileNorm(五筒)=%d want 1", got)
	}
}

func TestAkaRemoveTilesNormList(t *testing.T) {
	hand := tiles(0, 4, 34, 13)
	removed, rest, ok := removeTilesNormList(hand, []Tile{4, 4})
	if !ok {
		t.Fatalf("应能按 Base 移除2张五万")
	}
	if len(removed) != 2 {
		t.Fatalf("移除 %d 张 want 2", len(removed))
	}
	gotAka := false
	for _, x := range removed {
		if x.IsAka() {
			gotAka = true
		}
	}
	if !gotAka {
		t.Fatalf("移除的应含红五万原值: %v", removed)
	}
	if len(rest) != 2 || rest[0] != 0 || rest[1] != 13 {
		t.Fatalf("剩余 %v want [0 13]", rest)
	}
}
