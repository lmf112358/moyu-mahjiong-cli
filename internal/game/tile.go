package game

import (
	"fmt"
	"sort"
	"strings"
)

type Tile uint8

const NoTile Tile = 255

var tileNames = [34]string{
	"一", "二", "三", "四", "五", "六", "七", "八", "九",
	"①", "②", "③", "④", "⑤", "⑥", "⑦", "⑧", "⑨",
	"1", "2", "3", "4", "5", "6", "7", "8", "9",
	"東", "南", "西", "北", "白", "發", "中",
}

func (t Tile) String() string {
	if int(t) >= len(tileNames) {
		return "?"
	}
	return tileNames[t]
}

func (t Tile) Suit() int {
	if t < 9 {
		return 0
	}
	if t < 18 {
		return 1
	}
	if t < 27 {
		return 2
	}
	return 3
}

func (t Tile) Rank() int {
	if t >= 27 {
		return int(t) - 26
	}
	return int(t)%9 + 1
}

func (t Tile) IsHonor() bool    { return t >= 27 }
func (t Tile) IsTerminal() bool { return !t.IsHonor() && (t.Rank() == 1 || t.Rank() == 9) }
func (t Tile) IsYaochu() bool   { return t.IsHonor() || t.IsTerminal() }

func SortTiles(ts []Tile) { sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] }) }

func TilesString(ts []Tile) string {
	var b strings.Builder
	for _, t := range ts {
		b.WriteString(t.String())
	}
	return b.String()
}

func IndexedTiles(ts []Tile) string {
	var top, bottom strings.Builder
	for i, t := range ts {
		top.WriteString(fmt.Sprintf("%-3d", i+1))
		bottom.WriteString(fmt.Sprintf("%-3s", t.String()))
	}
	return strings.TrimRight(top.String(), " ") + "\n" + strings.TrimRight(bottom.String(), " ")
}

func FullWall(players int) []Tile {
	wall := make([]Tile, 0, 136)
	for t := Tile(0); t < 34; t++ {
		if players == 3 && t >= 1 && t <= 7 {
			continue
		}
		for i := 0; i < 4; i++ {
			wall = append(wall, t)
		}
	}
	return wall
}

func DoraFrom(ind Tile) Tile {
	switch {
	case ind < 27:
		base := ind / 9 * 9
		return base + (ind-base+1)%9
	case ind >= 27 && ind <= 30:
		return 27 + (ind-27+1)%4
	default:
		return 31 + (ind-31+1)%3
	}
}
