package game

import (
	"math"
	"sort"
	"strings"
)

type WinContext struct {
	Winner, From                     int
	WinTile                          Tile
	Tsumo, Riichi, Ippatsu           bool
	Dealer                           bool
	SeatWind, RoundWind              Tile
	DoraIndicators, UraIndicators    []Tile
	Rinshan, Haitei, Houtei, Chankan bool
	Players                          int
}

type ScoreResult struct {
	Yaku             []string   `json:"yaku"`
	YakuItems        []YakuItem `json:"yakuItems"`
	Han, Fu, Yakuman int
	Base             int
	Ron              int
	TsumoDealer      int
	TsumoChild       int
}

func EvaluateWin(p Player, ctx WinContext) (ScoreResult, bool) {
	shapes := WinningShapes(p.Hand, len(p.Melds))
	best := ScoreResult{}
	for _, shape := range shapes {
		r := evaluateShape(p, shape, ctx)
		if r.Yakuman > best.Yakuman || (r.Yakuman == best.Yakuman && (r.Han > best.Han || r.Han == best.Han && r.Fu > best.Fu)) {
			best = r
		}
	}
	if best.Yakuman == 0 && best.Han == 0 {
		return best, false
	}
	calculatePoints(&best, ctx.Dealer)
	return best, true
}

func evaluateShape(p Player, s Shape, c WinContext) ScoreResult {
	r := ScoreResult{}
	add := func(name string, han int) {
		r.Yaku = append(r.Yaku, name)
		r.YakuItems = append(r.YakuItems, YakuItem{Name: name, Han: han})
		r.Han += han
	}
	yakuman := func(name string) {
		r.Yaku = append(r.Yaku, name)
		r.YakuItems = append(r.YakuItems, YakuItem{Name: name, Yakuman: 1})
		r.Yakuman++
	}
	closed := true
	for _, m := range p.Melds {
		if m.Kind != Ankan {
			closed = false
		}
	}
	all := append([]Tile(nil), p.Hand...)
	for _, m := range p.Melds {
		all = append(all, m.Tiles...)
	}
	akaHan := 0
	for i, t := range all {
		if t.IsAka() {
			akaHan++
			all[i] = t.Base()
		}
	}
	if s.Kokushi {
		yakuman("国士无双")
		return r
	}
	triplets := []Tile{}
	sequences := []Tile{}
	for _, g := range s.Groups {
		if g.Sequence {
			sequences = append(sequences, g.Tile)
		} else {
			triplets = append(triplets, g.Tile)
		}
	}
	for _, m := range p.Melds {
		if m.Kind == Chi {
			sequences = append(sequences, minTile(m.Tiles))
		} else {
			triplets = append(triplets, m.Tiles[0])
		}
	}
	if isAllTripletsAndConcealed(s, p, c) {
		yakuman("四暗刻")
	}
	if containsTriplets(triplets, 31, 32, 33) {
		yakuman("大三元")
	}
	winds := 0
	for _, t := range []Tile{27, 28, 29, 30} {
		if contains(triplets, t) {
			winds++
		}
	}
	if winds == 4 {
		yakuman("大四喜")
	} else if winds == 3 && s.Pair >= 27 && s.Pair <= 30 {
		yakuman("小四喜")
	}
	if allMatch(all, func(t Tile) bool { return t.IsHonor() }) {
		yakuman("字一色")
	}
	if allMatch(all, func(t Tile) bool { return t.IsTerminal() }) {
		yakuman("清老头")
	}
	if r.Yakuman > 0 {
		return r
	}
	if c.Riichi && closed {
		add("立直", 1)
	}
	if c.Ippatsu && c.Riichi {
		add("一发", 1)
	}
	if c.Tsumo && closed {
		add("门前清自摸和", 1)
	}
	if c.Rinshan {
		add("岭上开花", 1)
	}
	if c.Chankan {
		add("抢杠", 1)
	}
	if c.Haitei {
		add("海底摸月", 1)
	}
	if c.Houtei {
		add("河底捞鱼", 1)
	}
	if allMatch(all, func(t Tile) bool { return !t.IsYaochu() }) {
		add("断幺九", 1)
	}
	if s.SevenPairs {
		add("七对子", 2)
		r.Fu = 25
	}
	for _, t := range triplets {
		if t >= 31 {
			add("役牌 "+t.String(), 1)
		}
		if t == c.SeatWind {
			add("自风 "+t.String(), 1)
		}
		if t == c.RoundWind {
			add("场风 "+t.String(), 1)
		}
	}
	if !s.SevenPairs && len(sequences) == 0 {
		add("对对和", 2)
	}
	concealedTrip := 0
	for _, g := range s.Groups {
		if !g.Sequence {
			concealedTrip++
		}
	}
	for _, m := range p.Melds {
		if m.Kind == Ankan {
			concealedTrip++
		}
	}
	if concealedTrip >= 3 {
		add("三暗刻", 2)
	}
	if allMatch(all, func(t Tile) bool { return t.IsYaochu() }) {
		add("混老头", 2)
	}
	if dragonTriplets(triplets) == 2 && s.Pair >= 31 {
		add("小三元", 2)
	}
	suits := map[int]bool{}
	honors := false
	for _, t := range all {
		if t.IsHonor() {
			honors = true
		} else {
			suits[t.Suit()] = true
		}
	}
	if len(suits) == 1 {
		if honors {
			if closed {
				add("混一色", 3)
			} else {
				add("混一色", 2)
			}
		} else {
			if closed {
				add("清一色", 6)
			} else {
				add("清一色", 5)
			}
		}
	}
	if hasSanshoku(sequences) {
		if closed {
			add("三色同顺", 2)
		} else {
			add("三色同顺", 1)
		}
	}
	if hasIttsu(sequences) {
		if closed {
			add("一气通贯", 2)
		} else {
			add("一气通贯", 1)
		}
	}
	if closed && isPinfu(s, sequences, c) {
		add("平和", 1)
		if c.Tsumo {
			r.Fu = 20
		}
	}
	if closed && hasIipeikou(s.Groups) {
		add("一杯口", 1)
	}
	// Dora are bonuses and cannot create a valid hand by themselves.
	baseHan := r.Han
	doraHan := 0
	for _, ind := range c.DoraIndicators {
		doraHan += countTile(all, DoraFrom(ind))
	}
	r.Han += doraHan + akaHan
	uraHan := 0
	if c.Riichi {
		for _, ind := range c.UraIndicators {
			uraHan += countTile(all, DoraFrom(ind))
		}
		r.Han += uraHan
	}
	if p.Kita > 0 {
		r.Han += p.Kita
	}
	if doraHan > 0 {
		r.Yaku = append(r.Yaku, "宝牌 "+itoa(doraHan))
		r.YakuItems = append(r.YakuItems, YakuItem{Name: "宝牌", Han: doraHan})
	}
	if akaHan > 0 {
		r.Yaku = append(r.Yaku, "赤宝牌 "+itoa(akaHan))
		r.YakuItems = append(r.YakuItems, YakuItem{Name: "赤宝牌", Han: akaHan})
	}
	if uraHan > 0 {
		r.Yaku = append(r.Yaku, "里宝牌 "+itoa(uraHan))
		r.YakuItems = append(r.YakuItems, YakuItem{Name: "里宝牌", Han: uraHan})
	}
	if p.Kita > 0 {
		r.Yaku = append(r.Yaku, "拔北 "+itoa(p.Kita))
		r.YakuItems = append(r.YakuItems, YakuItem{Name: "拔北", Han: p.Kita})
	}
	if baseHan == 0 {
		r.Han = 0
		r.Yaku = nil
		r.YakuItems = nil
		return r
	}
	if r.Fu == 0 {
		r.Fu = calculateFu(p, s, c, closed)
	}
	return r
}

func LimitName(r ScoreResult) string {
	if r.Yakuman > 1 {
		return itoa(r.Yakuman) + "倍役满"
	}
	if r.Yakuman == 1 {
		return "役满"
	}
	switch {
	case r.Han >= 13:
		return "累计役满"
	case r.Han >= 11:
		return "三倍满"
	case r.Han >= 8:
		return "倍满"
	case r.Han >= 6:
		return "跳满"
	case r.Han >= 5 || r.Base >= 2000:
		return "满贯"
	default:
		return ""
	}
}

func calculateFu(p Player, s Shape, c WinContext, closed bool) int {
	if s.SevenPairs {
		return 25
	}
	fu := 20
	if c.Tsumo {
		fu += 2
	}
	if !c.Tsumo && closed {
		fu += 10
	}
	if s.Pair >= 31 {
		fu += 2
	}
	if s.Pair == c.SeatWind {
		fu += 2
	}
	if s.Pair == c.RoundWind {
		fu += 2
	}
	for _, g := range s.Groups {
		if !g.Sequence {
			v := 2
			if g.Tile.IsYaochu() {
				v *= 2
			}
			v *= 2
			fu += v
		}
	}
	for _, m := range p.Melds {
		if m.Kind == Chi {
			continue
		}
		v := 2
		if m.Tiles[0].IsYaochu() {
			v *= 2
		}
		if m.Kind == Ankan {
			v *= 2
		}
		if len(m.Tiles) == 4 {
			v *= 4
		}
		fu += v
	}
	// Closed waits (tanki/kanchan/penchan) add 2 fu.
	if s.Pair == c.WinTile {
		fu += 2
	} else {
		for _, g := range s.Groups {
			if !g.Sequence {
				continue
			}
			r := c.WinTile.Rank() - g.Tile.Rank()
			if r == 1 || (g.Tile.Rank() == 1 && r == 2) || (g.Tile.Rank() == 7 && r == 0) {
				fu += 2
				break
			}
		}
	}
	if fu == 20 && !c.Tsumo {
		fu = 30
	}
	return int(math.Ceil(float64(fu)/10) * 10)
}

func calculatePoints(r *ScoreResult, dealer bool) {
	if r.Yakuman > 0 {
		r.Base = 8000 * r.Yakuman
	} else {
		base := r.Fu * (1 << (r.Han + 2))
		switch {
		case r.Han >= 13:
			base = 8000
		case r.Han >= 11:
			base = 6000
		case r.Han >= 8:
			base = 4000
		case r.Han >= 6:
			base = 3000
		case r.Han >= 5 || base >= 2000:
			base = 2000
		}
		r.Base = base
	}
	round100 := func(n int) int { return ((n + 99) / 100) * 100 }
	if dealer {
		r.Ron = round100(r.Base * 6)
		r.TsumoChild = round100(r.Base * 2)
		r.TsumoDealer = r.TsumoChild
	} else {
		r.Ron = round100(r.Base * 4)
		r.TsumoDealer = round100(r.Base * 2)
		r.TsumoChild = round100(r.Base)
	}
}

func isAllTripletsAndConcealed(s Shape, p Player, c WinContext) bool {
	if s.SevenPairs || s.Kokushi {
		return false
	}
	for _, g := range s.Groups {
		if g.Sequence {
			return false
		}
	}
	for _, m := range p.Melds {
		if m.Kind != Ankan {
			return false
		}
	}
	return c.Tsumo || s.Pair == c.WinTile
}
func contains(ts []Tile, t Tile) bool {
	for _, x := range ts {
		if x == t {
			return true
		}
	}
	return false
}
func containsTriplets(ts []Tile, want ...Tile) bool {
	for _, t := range want {
		if !contains(ts, t) {
			return false
		}
	}
	return true
}
func dragonTriplets(ts []Tile) int {
	n := 0
	for _, t := range []Tile{31, 32, 33} {
		if contains(ts, t) {
			n++
		}
	}
	return n
}
func minTile(ts []Tile) Tile {
	x := append([]Tile(nil), ts...)
	sort.Slice(x, func(i, j int) bool { return x[i] < x[j] })
	return x[0]
}
func allMatch(ts []Tile, f func(Tile) bool) bool {
	for _, t := range ts {
		if !f(t) {
			return false
		}
	}
	return true
}
func hasSanshoku(s []Tile) bool {
	for r := 0; r <= 6; r++ {
		if contains(s, Tile(r)) && contains(s, Tile(9+r)) && contains(s, Tile(18+r)) {
			return true
		}
	}
	return false
}
func hasIttsu(s []Tile) bool {
	for b := 0; b < 27; b += 9 {
		if contains(s, Tile(b)) && contains(s, Tile(b+3)) && contains(s, Tile(b+6)) {
			return true
		}
	}
	return false
}
func hasIipeikou(gs []Group) bool {
	m := map[Tile]int{}
	for _, g := range gs {
		if g.Sequence {
			m[g.Tile]++
		}
	}
	for _, n := range m {
		if n >= 2 {
			return true
		}
	}
	return false
}
func isPinfu(s Shape, seq []Tile, c WinContext) bool {
	if s.SevenPairs || len(seq) != 4 {
		return false
	}
	if s.Pair >= 31 || s.Pair == c.SeatWind || s.Pair == c.RoundWind {
		return false
	}
	if s.Pair == c.WinTile {
		return false
	}
	for _, g := range s.Groups {
		if !g.Sequence {
			continue
		}
		r := c.WinTile.Rank() - g.Tile.Rank()
		if r == 1 || (g.Tile.Rank() == 1 && r == 2) || (g.Tile.Rank() == 7 && r == 0) {
			return false
		}
	}
	return true
}
func itoa(n int) string {
	const d = "0123456789"
	if n < 10 {
		return string(d[n])
	}
	var b strings.Builder
	for n > 0 {
		b.WriteByte(d[n%10])
		n /= 10
	}
	x := []byte(b.String())
	for i, j := 0, len(x)-1; i < j; i, j = i+1, j-1 {
		x[i], x[j] = x[j], x[i]
	}
	return string(x)
}
