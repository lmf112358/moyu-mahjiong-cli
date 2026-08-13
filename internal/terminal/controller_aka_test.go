package terminal

import (
	"strings"
	"testing"

	"github.com/lmf112358/moyu-mahjiong-cli/internal/game"
)

func TestNormalCardsAkaRed(t *testing.T) {
	got := normalCards([]game.Tile{game.Aka5Man, 4}, 2, false, false)
	if !strings.Contains(got, colorAka) {
		t.Fatalf("红五牌块应染红:\n%s", got)
	}
	// 普通五万不应染红
	plain := normalCards([]game.Tile{4}, 1, false, false)
	if strings.Contains(plain, colorAka) {
		t.Fatalf("普通牌不应染红:\n%s", plain)
	}
}
