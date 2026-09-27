package app

import (
	"testing"

	"github.com/tomj59/dozer/internal/layout"
)

func TestNeighbor(t *testing.T) {
	root, _ := layout.Parse("2,2,1")
	s := layout.Solve(root, 100, 30, layout.Min{W: 10, H: 3}).Slots
	// [0][1] / [2][3] / [4]
	cases := []struct{ from, dx, dy, want int }{
		{0, 1, 0, 1}, {1, -1, 0, 0}, {0, 0, 1, 2}, {1, 0, 1, 3},
		{2, 0, 1, 4}, {3, 0, 1, 4}, {4, 0, -1, 2}, {0, 0, -1, -1}, {1, 1, 0, -1},
	}
	for _, c := range cases {
		if got := neighbor(s, c.from, c.dx, c.dy); got != c.want {
			t.Errorf("neighbor(%d, %d,%d) = %d, want %d", c.from, c.dx, c.dy, got, c.want)
		}
	}
}

func TestFollow(t *testing.T) {
	cases := []struct{ off, pos, size, screen, canvas, want int }{
		{0, 0, 20, 80, 60, 0},   // canvas fits
		{0, 42, 20, 50, 62, 12}, // slot off the right edge
		{12, 0, 20, 50, 62, 0},  // slot off the left edge
		{5, 10, 20, 50, 62, 5},  // already visible: don't move
	}
	for _, c := range cases {
		if got := follow(c.off, c.pos, c.size, c.screen, c.canvas); got != c.want {
			t.Errorf("follow(%+v) = %d, want %d", c, got, c.want)
		}
	}
}

func TestDropPaste(t *testing.T) {
	for in, want := range map[string]string{
		"jk":                             "jk",
		"a\x1b[200~yq\x1b[201~b":         "ab",
		"\x1b[200~x\x1b[201~\x1b[200~y":  "",
		"k\x1b[200~one\x1b[201~j\x1b[A": "kj\x1b[A",
	} {
		if got := string(dropPaste([]byte(in))); got != want {
			t.Errorf("dropPaste(%q) = %q, want %q", in, got, want)
		}
	}
}
