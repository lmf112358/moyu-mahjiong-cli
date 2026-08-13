package game

import "testing"

// 立直后应锁定摸切、View 反映立直状态
func TestRiichiLocksToTsumoGiri(t *testing.T) {
	cap := &captureController{}
	e := NewEngine(Config{Players: 4}, []string{"p", "AI", "AI", "AI"}, []Controller{cap, AI{}, AI{}, AI{}})
	e.Players[0].Hand = tiles(0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 27, 27, 31, 31)
	e.Players[0].Riichi = true
	drawn := Tile(32)
	e.Players[0].Hand = append(e.Players[0].Hand, drawn)
	SortTiles(e.Players[0].Hand)
	e.drawDecision(0, drawn)

	discards := 0
	for _, o := range cap.decision.Options {
		if o.Action.Kind == ActDiscard {
			discards++
		}
	}
	if discards != 1 {
		t.Fatalf("立直后应只有1个摸切选项，实有 %d: %+v", discards, cap.decision.Options)
	}
	if !cap.decision.View.Players[0].Riichi {
		t.Fatal("View.Players[0].Riichi 应为 true（立直状态须显示）")
	}
}

// riichiPicker 总是选第一个立直选项
type riichiPicker struct{}

func (riichiPicker) Decide(d Decision) Action {
	for _, o := range d.Options {
		if o.Action.Kind == ActRiichi {
			return o.Action
		}
	}
	return Action{Kind: ActQuit}
}

// 立直宣言本身必须把 Riichi 置 true
func TestRiichiDeclarationSetsFlag(t *testing.T) {
	e := NewEngine(Config{Players: 4}, []string{"p", "AI", "AI", "AI"}, []Controller{riichiPicker{}, AI{}, AI{}, AI{}})
	e.Players[0].Hand = tiles(0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 27, 27, 31, 31)
	outcome, _ := e.drawDecision(0, 31)
	if outcome != ActDiscard {
		t.Fatalf("立直宣言后 outcome=%v want ActDiscard", outcome)
	}
	if !e.Players[0].Riichi {
		t.Fatal("立直宣言后 e.Players[0].Riichi 应为 true")
	}
}
