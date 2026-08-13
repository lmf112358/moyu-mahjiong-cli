package game

type MeldKind string

const (
	Chi   MeldKind = "吃"
	Pon   MeldKind = "碰"
	Kan   MeldKind = "杠"
	Ankan MeldKind = "暗杠"
)

type Meld struct {
	Kind  MeldKind `json:"kind"`
	Tiles []Tile   `json:"tiles"`
	From  int      `json:"from"`
}

type Player struct {
	Name    string `json:"name"`
	Hand    []Tile `json:"hand,omitempty"`
	Melds   []Meld `json:"melds"`
	River   []Tile `json:"river"`
	Score   int    `json:"score"`
	Riichi  bool   `json:"riichi"`
	Ippatsu bool   `json:"ippatsu"`
	Furiten bool   `json:"furiten"`
	Kita    int    `json:"kita"`
}

type ActionKind string

const (
	ActDiscard ActionKind = "discard"
	ActTsumo   ActionKind = "tsumo"
	ActRon     ActionKind = "ron"
	ActChi     ActionKind = "chi"
	ActPon     ActionKind = "pon"
	ActKan     ActionKind = "kan"
	ActRiichi  ActionKind = "riichi"
	ActKita    ActionKind = "kita"
	ActPass    ActionKind = "pass"
	ActQuit    ActionKind = "quit"
)

type Action struct {
	Kind  ActionKind `json:"kind"`
	Tile  Tile       `json:"tile"`
	Tiles []Tile     `json:"tiles,omitempty"`
	Index int        `json:"index,omitempty"`
}

type Option struct {
	Label  string `json:"label"`
	Action Action `json:"action"`
}

type View struct {
	Players      []Player `json:"players"`
	You          int      `json:"you"`
	Turn         int      `json:"turn"`
	Active       int      `json:"active"`
	Dealer       int      `json:"dealer"`
	RoundWind    Tile     `json:"roundWind"`
	HandNumber   int      `json:"handNumber"`
	Honba        int      `json:"honba"`
	RiichiSticks int      `json:"riichiSticks"`
	WallLeft     int      `json:"wallLeft"`
	Dora         []Tile   `json:"dora"`
	Message      string   `json:"message"`
	LastDiscard  Tile     `json:"lastDiscard"`
	LastFrom     int      `json:"lastFrom"`
	DrawnIndex   int      `json:"drawnIndex"`
	HasDrawn     bool     `json:"hasDrawn"`
}

type Decision struct {
	View    View     `json:"view"`
	Prompt  string   `json:"prompt"`
	Options []Option `json:"options"`
}

type Controller interface{ Decide(Decision) Action }

type YakuItem struct {
	Name    string `json:"name"`
	Han     int    `json:"han,omitempty"`
	Yakuman int    `json:"yakuman,omitempty"`
}

type ScoreChange struct {
	Player int    `json:"player"`
	Name   string `json:"name"`
	Before int    `json:"before"`
	Delta  int    `json:"delta"`
	After  int    `json:"after"`
}

type WinDetail struct {
	Winner     int        `json:"winner"`
	WinnerName string     `json:"winnerName"`
	From       int        `json:"from"`
	FromName   string     `json:"fromName,omitempty"`
	Tsumo      bool       `json:"tsumo"`
	WinTile    Tile       `json:"winTile"`
	Structure  string     `json:"structure"`
	Hand       []Tile     `json:"hand"`
	Melds      []Meld     `json:"melds"`
	Yaku       []YakuItem `json:"yaku"`
	Han        int        `json:"han"`
	Fu         int        `json:"fu"`
	Yakuman    int        `json:"yakuman"`
	Limit      string     `json:"limit"`
	Gain       int        `json:"gain"`
	Kita       int        `json:"kita"`
	Dora       []Tile     `json:"doraIndicators"`
	UraDora    []Tile     `json:"uraDoraIndicators,omitempty"`
}

type Settlement struct {
	RoundWind    Tile          `json:"roundWind"`
	HandNumber   int           `json:"handNumber"`
	Honba        int           `json:"honba"`
	RiichiSticks int           `json:"riichiSticks"`
	Wins         []WinDetail   `json:"wins"`
	Changes      []ScoreChange `json:"changes"`
}

type SettlementViewer interface {
	ShowSettlement(Settlement)
}

var WindNames = []string{"东", "南", "西", "北"}
