package game

import "testing"

func tiles(xs ...int) []Tile {
	r := make([]Tile, len(xs))
	for i, x := range xs {
		r[i] = Tile(x)
	}
	return r
}

func TestStandardAndSpecialHands(t *testing.T) {
	tests := []struct {
		name string
		hand []Tile
		want bool
	}{
		{"standard", tiles(0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 27, 27, 31, 31), true},
		{"seven pairs", tiles(0, 0, 8, 8, 9, 9, 17, 17, 18, 18, 27, 27, 31, 31), true},
		{"kokushi", tiles(0, 8, 9, 17, 18, 26, 27, 28, 29, 30, 31, 32, 33, 33), true},
		{"incomplete", tiles(0, 1, 3, 9, 10, 11, 18, 19, 20, 27, 27, 27, 31, 31), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsComplete(tt.hand, 0); got != tt.want {
				t.Fatalf("IsComplete=%v want %v", got, tt.want)
			}
		})
	}
}

func TestWaits(t *testing.T) {
	h := tiles(0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 27, 27, 31)
	w := Waits(h, 0, 4)
	if len(w) != 1 || w[0] != 31 {
		t.Fatalf("waits=%v", w)
	}
}

func TestDoraWrap(t *testing.T) {
	cases := map[Tile]Tile{8: 0, 17: 9, 26: 18, 30: 27, 33: 31}
	for in, want := range cases {
		if got := DoraFrom(in); got != want {
			t.Errorf("DoraFrom(%v)=%v want %v", in, got, want)
		}
	}
}

func TestSanmaManzuDoraWrap(t *testing.T) {
	if got := doraFrom(0, 3); got != 8 {
		t.Fatalf("sanma DoraFrom(一万)=%v want 九万", got)
	}
	if got := doraFrom(8, 3); got != 0 {
		t.Fatalf("sanma DoraFrom(九万)=%v want 一万", got)
	}
}

// Cases used by Poker-sang/Mahjong's sample program: thirteen-sided
// kokushi and the nine-sided pure-suit shape 1112345678999.
func TestReferenceReadyHandCases(t *testing.T) {
	yaochu := tiles(0, 8, 9, 17, 18, 26, 27, 28, 29, 30, 31, 32, 33)
	if got := Waits(yaochu, 0, 4); len(got) != 13 {
		t.Fatalf("kokushi waits=%v", got)
	}
	nineGates := tiles(0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 8, 8)
	got := Waits(nineGates, 0, 4)
	if len(got) != 9 {
		t.Fatalf("nine gates waits=%v", got)
	}
	for i, tile := range got {
		if tile != Tile(i) {
			t.Fatalf("wait %d=%v", i, tile)
		}
	}
}

func TestSanmaWall(t *testing.T) {
	w := FullWall(3)
	if len(w) != 108 {
		t.Fatalf("len=%d", len(w))
	}
	for _, x := range w {
		if x >= 1 && x <= 7 {
			t.Fatalf("found removed tile %v", x)
		}
	}
}

func TestScoreRiichiPinfu(t *testing.T) {
	p := Player{Hand: tiles(0, 1, 2, 3, 4, 5, 10, 11, 12, 22, 23, 24, 27, 27), Riichi: true}
	r, ok := EvaluateWin(p, WinContext{WinTile: 24, Riichi: true, SeatWind: 28, RoundWind: 27})
	if !ok || r.Han < 1 {
		t.Fatalf("score=%+v ok=%v", r, ok)
	}
}

func TestPinfuTsumoIsTwentyFu(t *testing.T) {
	p := Player{Hand: tiles(0, 1, 2, 3, 4, 5, 10, 11, 12, 22, 23, 24, 29, 29)}
	r, ok := EvaluateWin(p, WinContext{WinTile: 12, Tsumo: true, SeatWind: 28, RoundWind: 27})
	if !ok || r.Fu != 20 {
		t.Fatalf("score=%+v ok=%v", r, ok)
	}
	found := false
	for _, y := range r.YakuItems {
		if y.Name == "平和" && y.Han == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing pinfu detail: %+v", r.YakuItems)
	}
}

func TestPinfuWaitDoesNotMatchSequenceInAnotherSuit(t *testing.T) {
	p := Player{Hand: tiles(2, 3, 4, 12, 13, 14, 18, 19, 20, 24, 25, 26, 29, 29), Riichi: true}
	r, ok := EvaluateWin(p, WinContext{WinTile: 4, Riichi: true, SeatWind: 28, RoundWind: 27, Players: 4})
	if !ok || r.Han != 2 || r.Fu != 30 {
		t.Fatalf("cross-suit sequence changed pinfu scoring: score=%+v ok=%v", r, ok)
	}
}

func TestRedFiveMeldUsesBaseTileAsSequenceStart(t *testing.T) {
	if got := minTile([]Tile{Aka5Man, 5, 6}); got != 4 {
		t.Fatalf("red 567 sequence start=%v want 五万", got)
	}
}

func TestPointTable(t *testing.T) {
	tests := []struct {
		han, fu                  int
		dealer                   bool
		ron, dealerPay, childPay int
	}{
		{1, 30, false, 1000, 500, 300},
		{3, 40, false, 5200, 2600, 1300},
		{4, 40, false, 8000, 4000, 2000},
		{3, 30, true, 5800, 2000, 2000},
	}
	for _, tt := range tests {
		r := ScoreResult{Han: tt.han, Fu: tt.fu}
		calculatePoints(&r, tt.dealer)
		if r.Ron != tt.ron || r.TsumoDealer != tt.dealerPay || r.TsumoChild != tt.childPay {
			t.Errorf("%dhan %dfu dealer=%v => %+v", tt.han, tt.fu, tt.dealer, r)
		}
	}
}

func TestDoraAndUraAreSeparateHanItems(t *testing.T) {
	p := Player{Hand: tiles(0, 1, 2, 3, 4, 5, 10, 11, 12, 22, 23, 24, 29, 29), Riichi: true}
	r, ok := EvaluateWin(p, WinContext{WinTile: 12, Tsumo: true, Riichi: true, SeatWind: 28, RoundWind: 27, DoraIndicators: []Tile{8}, UraIndicators: []Tile{0}})
	if !ok {
		t.Fatal("expected valid hand")
	}
	want := map[string]int{"宝牌": 1, "里宝牌": 1}
	for _, y := range r.YakuItems {
		if _, exists := want[y.Name]; exists {
			want[y.Name] -= y.Han
		}
	}
	for name, remaining := range want {
		if remaining != 0 {
			t.Fatalf("%s detail incorrect: %+v", name, r.YakuItems)
		}
	}
}
