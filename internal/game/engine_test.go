package game

import "testing"

func TestAIMatchesFinish(t *testing.T) {
	for _, n := range []int{3, 4} {
		for seed := uint64(1); seed <= 8; seed++ {
			cs := make([]Controller, n)
			names := make([]string, n)
			for i := range cs {
				cs[i] = AI{Level: 1}
				names[i] = "AI"
			}
			e := NewEngine(Config{Players: n, Rounds: 1, Seed: seed}, names, cs)
			r := e.Run()
			if len(r.Players) != n {
				t.Fatal("wrong result size")
			}
			sum := 0
			for _, p := range r.Players {
				sum += p.Score
			}
			start := 25000 * n
			if n == 3 {
				start = 35000 * n
			}
			if sum != start {
				t.Fatalf("%d-player seed %d score sum=%d want %d", n, seed, sum, start)
			}
		}
	}
}

func TestOnlyConcealedMeldsAllowRiichi(t *testing.T) {
	if isClosed(Player{Melds: []Meld{{Kind: Kan}}}) {
		t.Fatal("open kan must break menzen")
	}
	if !isClosed(Player{Melds: []Meld{{Kind: Ankan}}}) {
		t.Fatal("concealed kan must preserve menzen")
	}
}

type captureController struct {
	decision   Decision
	settlement Settlement
}

func (c *captureController) Decide(d Decision) Action {
	c.decision = d
	return Action{Kind: ActQuit}
}

func (c *captureController) ShowSettlement(s Settlement) { c.settlement = s }

func TestRiichiOptionGeneration(t *testing.T) {
	has := func(options []Option, kind ActionKind) bool {
		for _, o := range options {
			if o.Action.Kind == kind {
				return true
			}
		}
		return false
	}
	closedCapture := &captureController{}
	e := NewEngine(Config{Players: 4}, []string{"p"}, []Controller{closedCapture, AI{}, AI{}, AI{}})
	e.Players[0].Hand = tiles(0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 27, 27, 31, 31)
	e.drawDecision(0, 31)
	if !has(closedCapture.decision.Options, ActRiichi) {
		t.Fatal("closed tenpai hand must offer riichi")
	}

	openCapture := &captureController{}
	e.Controllers[0] = openCapture
	e.Players[0].Hand = tiles(0, 1, 2, 9, 10, 11, 27, 27, 27, 31, 31)
	e.Players[0].Melds = []Meld{{Kind: Kan, Tiles: tiles(8, 8, 8, 8)}}
	e.drawDecision(0, 31)
	if has(openCapture.decision.Options, ActRiichi) {
		t.Fatal("open tenpai hand must not offer riichi")
	}
}

func TestWinProducesStructuredSettlement(t *testing.T) {
	capture := &captureController{}
	e := NewEngine(Config{Players: 4}, []string{"赢家", "乙", "丙", "丁"}, []Controller{capture, AI{}, AI{}, AI{}})
	e.Players[0].Hand = tiles(0, 1, 2, 3, 4, 5, 10, 11, 12, 22, 23, 24, 29, 29)
	e.win([]int{0}, 0, true, 12, false)
	if len(capture.settlement.Wins) != 1 {
		t.Fatalf("settlement=%+v", capture.settlement)
	}
	w := capture.settlement.Wins[0]
	if w.WinnerName != "赢家" || w.Fu != 20 || w.Han < 2 || w.Structure != "四面子一雀头" || len(w.Yaku) == 0 {
		t.Fatalf("win detail=%+v", w)
	}
	if len(capture.settlement.Changes) != 4 {
		t.Fatalf("changes=%+v", capture.settlement.Changes)
	}
}
