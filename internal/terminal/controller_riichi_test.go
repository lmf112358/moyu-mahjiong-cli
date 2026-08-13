package terminal

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/lmf112358/moyu-mahjiong-cli/internal/game"
)

func TestRiichiRenderedBothModes(t *testing.T) {
	for _, mode := range []DisplayMode{StealthMode, NormalMode} {
		var buf bytes.Buffer
		c := &Controller{Out: &buf, In: bufio.NewReader(strings.NewReader("")), Mode: mode}
		c.render(game.Decision{View: game.View{
			Players:   []game.Player{{Name: "me", Score: 25000, Riichi: true, Hand: []game.Tile{0}, River: []game.Tile{}}},
			You:       0,
			Dealer:    0,
			Active:    0,
			RoundWind: 27,
		}})
		if !strings.Contains(buf.String(), "立直") {
			t.Errorf("%v 模式未显示立直:\n%s", mode, buf.String())
		}
	}
}
