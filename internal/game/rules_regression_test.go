package game

import "testing"

type actionKindPicker struct {
	want      ActionKind
	decisions []Decision
}

func (c *actionKindPicker) Decide(d Decision) Action {
	c.decisions = append(c.decisions, d)
	for _, o := range d.Options {
		if o.Action.Kind == c.want {
			return o.Action
		}
	}
	for _, o := range d.Options {
		if o.Action.Kind == ActPass {
			return o.Action
		}
	}
	return Action{Kind: ActQuit}
}

func TestSanmaDoesNotOfferChi(t *testing.T) {
	controllers := []*actionKindPicker{{want: ActPass}, {want: ActChi}, {want: ActPass}}
	e := NewEngine(Config{Players: 3}, []string{"a", "b", "c"}, []Controller{controllers[0], controllers[1], controllers[2]})
	e.Players[1].Hand = tiles(9, 11)
	_, caller, _, _ := e.reactions(0, 10)
	if caller >= 0 {
		t.Fatalf("sanma chi caller=%d, want none", caller)
	}
	for _, d := range controllers[1].decisions {
		for _, o := range d.Options {
			if o.Action.Kind == ActChi {
				t.Fatalf("sanma offered chi: %+v", d.Options)
			}
		}
	}
}

func TestDaiminkanReplacementCanTsumoAndSetsRinshan(t *testing.T) {
	picker := &actionKindPicker{want: ActTsumo}
	e := NewEngine(Config{Players: 4}, []string{"a", "b", "c", "d"}, []Controller{AI{}, picker, AI{}, AI{}})
	e.wall = make([]Tile, 20)
	e.dead = make([]Tile, 14)
	e.dead[0] = 31
	e.Players[0].River = []Tile{27}
	e.Players[1].Hand = tiles(0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 27, 27, 31)

	replacement := e.call(1, 0, Kan, 27, []Tile{27, 27, 27})
	if replacement != 31 {
		t.Fatalf("replacement=%v want 白", replacement)
	}
	outcome, winTile, ok := e.resolveDrawDecision(1, replacement)
	if !ok || outcome != ActTsumo || winTile != 31 {
		t.Fatalf("outcome=%v tile=%v ok=%v, want rinshan tsumo", outcome, winTile, ok)
	}
	ctx := e.context(1, 1, true, replacement)
	if !ctx.Rinshan || ctx.Haitei {
		t.Fatalf("rinshan context=%+v", ctx)
	}
}

func TestIppatsuSurvivesUntilRiichiPlayersNextDiscard(t *testing.T) {
	e := NewEngine(Config{Players: 4}, nil, []Controller{AI{}, AI{}, AI{}, AI{}})
	e.Players[0].Riichi = true
	e.Players[0].Ippatsu = true
	e.Players[0].Hand = tiles(1, 2)
	e.Players[1].Hand = tiles(0)

	e.discard(1, 0)
	if !e.Players[0].Ippatsu {
		t.Fatal("opponent's ordinary discard cleared ippatsu before reactions")
	}
	e.discard(0, 1)
	if e.Players[0].Ippatsu {
		t.Fatal("riichi player's next discard must end ippatsu")
	}
}

func TestTemporaryAndRiichiFuritenLifetime(t *testing.T) {
	e := NewEngine(Config{Players: 4}, nil, []Controller{AI{}, AI{}, AI{}, AI{}})
	e.wall = tiles(0, 1, 2)
	e.missedRon[0] = true
	e.Players[0].Furiten = true

	e.drawLive(1)
	if !e.missedRon[0] {
		t.Fatal("another player's draw cleared temporary furiten")
	}
	e.drawLive(0)
	if e.missedRon[0] || e.Players[0].Furiten {
		t.Fatal("own draw did not clear temporary furiten")
	}

	e.Players[0].Riichi = true
	e.missedRon[0] = true
	e.drawLive(0)
	if !e.missedRon[0] || !e.Players[0].Furiten {
		t.Fatal("riichi furiten must persist after own draw")
	}
}

func TestDiscardedRedFiveCausesFuritenOnBaseFive(t *testing.T) {
	e := NewEngine(Config{Players: 4}, nil, []Controller{AI{}, AI{}, AI{}, AI{}})
	e.Players[0].Hand = tiles(0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 27, 27, 4)
	e.Players[0].River = []Tile{Aka5Man}
	if !e.isFuriten(0) {
		t.Fatal("discarded red five must cause furiten when waiting on base five")
	}
}
