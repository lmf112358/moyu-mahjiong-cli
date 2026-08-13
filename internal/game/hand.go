package game

import "sort"

type Group struct {
	Sequence bool
	Tile     Tile
}

type Shape struct {
	Pair       Tile
	Groups     []Group
	SevenPairs bool
	Kokushi    bool
}

func Counts(tiles []Tile) [34]int {
	var c [34]int
	for _, t := range tiles {
		if b := t.Base(); b < 34 {
			c[b]++
		}
	}
	return c
}

func WinningShapes(hand []Tile, openMelds int) []Shape {
	c := Counts(hand)
	needGroups := 4 - openMelds
	var out []Shape
	if openMelds == 0 && len(hand) == 14 {
		pairs, distinct := 0, 0
		yaochu := []Tile{0, 8, 9, 17, 18, 26, 27, 28, 29, 30, 31, 32, 33}
		kokushi, pair := true, false
		for _, t := range yaochu {
			if c[t] == 0 {
				kokushi = false
			}
			if c[t] >= 2 {
				pair = true
			}
		}
		for _, n := range c {
			if n > 0 {
				distinct++
			}
			if n == 2 {
				pairs++
			}
		}
		if kokushi && pair && distinct == 13 {
			out = append(out, Shape{Kokushi: true})
		}
		if pairs == 7 {
			out = append(out, Shape{SevenPairs: true})
		}
	}
	if len(hand) != needGroups*3+2 {
		return out
	}
	for p := Tile(0); p < 34; p++ {
		if c[p] < 2 {
			continue
		}
		c[p] -= 2
		walkGroups(&out, c, needGroups, Shape{Pair: p})
		c[p] += 2
	}
	return out
}

func walkGroups(out *[]Shape, c [34]int, need int, s Shape) {
	if len(s.Groups) == need {
		for _, n := range c {
			if n != 0 {
				return
			}
		}
		copyGroups := append([]Group(nil), s.Groups...)
		s.Groups = copyGroups
		*out = append(*out, s)
		return
	}
	first := -1
	for i, n := range c {
		if n > 0 {
			first = i
			break
		}
	}
	if first < 0 {
		return
	}
	if c[first] >= 3 {
		n := c
		n[first] -= 3
		walkGroups(out, n, need, Shape{Pair: s.Pair, Groups: append(s.Groups, Group{Tile: Tile(first)})})
	}
	if first < 27 && first%9 <= 6 && c[first+1] > 0 && c[first+2] > 0 {
		n := c
		n[first]--
		n[first+1]--
		n[first+2]--
		walkGroups(out, n, need, Shape{Pair: s.Pair, Groups: append(s.Groups, Group{Sequence: true, Tile: Tile(first)})})
	}
}

func IsComplete(hand []Tile, melds int) bool { return len(WinningShapes(hand, melds)) > 0 }

func Waits(hand []Tile, melds int, players int) []Tile {
	var waits []Tile
	for t := Tile(0); t < 34; t++ {
		if players == 3 && t >= 1 && t <= 7 {
			continue
		}
		if countTileNorm(hand, t) >= 4 {
			continue
		}
		x := append(append([]Tile(nil), hand...), t)
		if IsComplete(x, melds) {
			waits = append(waits, t)
		}
	}
	return waits
}

func TenpaiDiscards(hand []Tile, melds int, players int) []int {
	seen := map[Tile]bool{}
	var result []int
	for i, t := range hand {
		if seen[t] {
			continue
		}
		seen[t] = true
		x := append([]Tile(nil), hand[:i]...)
		x = append(x, hand[i+1:]...)
		if len(Waits(x, melds, players)) > 0 {
			result = append(result, i)
		}
	}
	return result
}

func countTile(ts []Tile, t Tile) int {
	n := 0
	for _, x := range ts {
		if x == t {
			n++
		}
	}
	return n
}

func countTileNorm(ts []Tile, t Tile) int {
	n := 0
	b := t.Base()
	for _, x := range ts {
		if x.Base() == b {
			n++
		}
	}
	return n
}

func removeTiles(hand []Tile, wanted ...Tile) ([]Tile, bool) {
	x := append([]Tile(nil), hand...)
	for _, t := range wanted {
		found := -1
		for i, v := range x {
			if v == t {
				found = i
				break
			}
		}
		if found < 0 {
			return hand, false
		}
		x = append(x[:found], x[found+1:]...)
	}
	SortTiles(x)
	return x, true
}

func removeTilesNormList(hand []Tile, wanted []Tile) ([]Tile, []Tile, bool) {
	x := append([]Tile(nil), hand...)
	removed := make([]Tile, 0, len(wanted))
	for _, wt := range wanted {
		found := -1
		for i, v := range x {
			if v.Base() == wt.Base() {
				found = i
				break
			}
		}
		if found < 0 {
			return nil, hand, false
		}
		removed = append(removed, x[found])
		x = append(x[:found], x[found+1:]...)
	}
	SortTiles(x)
	return removed, x, true
}

func uniqueTiles(ts []Tile) []Tile {
	seen := map[Tile]bool{}
	out := []Tile{}
	for _, t := range ts {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
