package game

import "testing"

type quitController struct{}

func (quitController) Decide(Decision) Action { return Action{Kind: ActQuit} }

func TestQuitStopsMatch(t *testing.T) {
	cs := []Controller{quitController{}, AI{Level: 1}, AI{Level: 1}}
	e := NewEngine(Config{Players: 3, Rounds: 2, Seed: 7}, []string{"quit", "a", "b"}, cs)
	r := e.Run()
	if !e.aborted {
		t.Fatal("match was not aborted")
	}
	if len(r.Log) > 4 {
		t.Fatalf("quit should stop immediately, log=%v", r.Log)
	}
}
