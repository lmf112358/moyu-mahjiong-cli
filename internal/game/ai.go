package game

import "math/rand/v2"

type AI struct{ Level int }

func (a AI) Decide(d Decision) Action {
	if len(d.Options) == 0 {
		return Action{Kind: ActQuit}
	}
	for _, k := range []ActionKind{ActTsumo, ActRon, ActRiichi, ActKita, ActKan} {
		for _, o := range d.Options {
			if o.Action.Kind == k {
				return o.Action
			}
		}
	}
	// Prefer useful calls, but avoid opening a hand for isolated terminals/honors.
	for _, k := range []ActionKind{ActPon, ActChi} {
		for _, o := range d.Options {
			if o.Action.Kind == k && a.Level > 0 {
				return o.Action
			}
		}
	}
	best := -1 << 30
	choices := []Action{}
	for _, o := range d.Options {
		if o.Action.Kind != ActDiscard {
			continue
		}
		score := discardScore(d.View.Players[d.View.You].Hand, o.Action.Tile)
		if score > best {
			best = score
			choices = []Action{o.Action}
		} else if score == best {
			choices = append(choices, o.Action)
		}
	}
	if len(choices) > 0 {
		return choices[rand.IntN(len(choices))]
	}
	for _, o := range d.Options {
		if o.Action.Kind == ActPass {
			return o.Action
		}
	}
	return d.Options[len(d.Options)-1].Action
}

func discardScore(hand []Tile, t Tile) int {
	// Higher means more disposable: isolated honors/terminals first, then isolated simples.
	n := countTile(hand, t)
	score := 0
	if t.IsHonor() {
		score += 35
		if n >= 2 {
			score -= 45
		}
	} else {
		r := t.Rank()
		if r == 1 || r == 9 {
			score += 18
		}
		near := 0
		for _, x := range hand {
			if x.Suit() == t.Suit() {
				d := x.Rank() - r
				if d < 0 {
					d = -d
				}
				if d > 0 && d <= 2 {
					near++
				}
			}
		}
		score -= near * 12
		if n >= 2 {
			score -= 25
		}
	}
	return score
}
