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

func TestStealthGroupedFormat(t *testing.T) {
	// 五筒x2 + 红五筒(35) + 三索(20) + 五索(22) + 东x2(27)
	got := stealthGrouped([]game.Tile{13, 13, 35, 20, 22, 27, 27})
	t.Logf("分组输出: [%s]", got)
	for _, want := range []string{"筒:", "索:", "字:"} {
		if !strings.Contains(got, want) {
			t.Errorf("缺 %s 分组: %s", want, got)
		}
	}
}
