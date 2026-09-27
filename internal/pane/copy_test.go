package pane

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

// fakeSrc is text lines numbered from first; wraps marks soft-wrapped lines.
type fakeSrc struct {
	first      int
	lines      []string
	wraps      map[int]bool
	rows, cols int
}

func (f fakeSrc) Line(n int) (uv.Line, bool) {
	i := n - f.first
	if i < 0 || i >= len(f.lines) {
		return nil, false
	}
	var l uv.Line
	for _, r := range f.lines[i] {
		w := 1
		if r >= 0x3000 {
			w = 2
		}
		l = append(l, uv.Cell{Content: string(r), Width: w})
		if w == 2 {
			l = append(l, uv.Cell{})
		}
	}
	return l, f.wraps[n]
}
func (f fakeSrc) First() int { return f.first }
func (f fakeSrc) Last() int  { return f.first + len(f.lines) - 1 }
func (f fakeSrc) Rows() int  { return f.rows }
func (f fakeSrc) Cols() int  { return f.cols }

func src(lines ...string) fakeSrc {
	return fakeSrc{first: 100, lines: lines, rows: 3, cols: 20}
}

func TestCopyScrollAndClamp(t *testing.T) {
	s := src("a", "b", "c", "d", "e", "f") // live screen = lines 103..105
	c := NewCopyMode(103, 0, 2)            // cursor on "f"
	if c.Scrolled(s) != 0 {
		t.Fatalf("scrolled = %d", c.Scrolled(s))
	}
	c.Scroll(s, -2)
	if c.Top != 101 || c.Y != 103 || c.Scrolled(s) != 2 {
		t.Fatalf("after scroll: top=%d y=%d", c.Top, c.Y)
	}
	c.Scroll(s, -10) // can't go past the oldest line
	if c.Top != 100 {
		t.Fatalf("top = %d, want 100", c.Top)
	}
	c.Scroll(s, 50) // nor below the live screen
	if c.Top != 103 {
		t.Fatalf("top = %d, want 103", c.Top)
	}
	c.Move(s, 0, -5) // cursor drags the view along
	if c.Y != 100 || c.Top != 100 {
		t.Fatalf("y=%d top=%d", c.Y, c.Top)
	}
}

func TestCopySelectionText(t *testing.T) {
	s := src("hello world   ", "second line", "third")
	c := NewCopyMode(100, 6, 0)
	c.Select(SelChar)
	c.Move(s, 0, 1)
	c.Move(s, -1, 0) // x = 5: "second" ends at 5
	if got := c.Text(s); got != "world\nsecond" {
		t.Fatalf("char selection = %q", got)
	}
	c.Select(SelLine)
	if got := c.Text(s); got != "hello world\nsecond line" {
		t.Fatalf("line selection = %q", got)
	}
	// Backwards selection is normalized.
	c = NewCopyMode(100, 2, 1)
	c.Select(SelChar)
	c.Move(s, 0, -1)
	if got := c.Text(s); got != "llo world\nsec" {
		t.Fatalf("backwards selection = %q", got)
	}
}

func TestCopyJoinsSoftWraps(t *testing.T) {
	s := src("a long line tha", "t wrapped", "next")
	s.wraps = map[int]bool{100: true}
	c := NewCopyMode(100, 0, 0)
	c.Select(SelLine)
	c.Move(s, 0, 2)
	if got := c.Text(s); got != "a long line that wrapped\nnext" {
		t.Fatalf("got %q", got)
	}
}

func TestCopyWideGlyphs(t *testing.T) {
	s := src("日本語 ok")
	c := NewCopyMode(100, 2, 0) // on 本
	c.Select(SelChar)
	c.LineEnd(s)
	if got := c.Text(s); got != "本語 ok" {
		t.Fatalf("got %q", got)
	}
}

func TestCopySearch(t *testing.T) {
	s := src("error one", "fine", "ERROR two", "fine", "error three")
	c := NewCopyMode(102, 0, 2) // bottom line
	if !c.Search(s, "error", true, false) || c.Y != 102 || c.X != 0 {
		t.Fatalf("backwards search: y=%d x=%d", c.Y, c.X)
	}
	if !c.Search(s, "", false, false) || c.Y != 100 { // n: continues backwards
		t.Fatalf("n: y=%d", c.Y)
	}
	if !c.Search(s, "", false, true) || c.Y != 102 { // N: reverses
		t.Fatalf("N: y=%d", c.Y)
	}
	if !c.Search(s, "ERROR", false, false) || c.Y != 102 {
		// Capitals make it case-sensitive; from 102 forward wraps to 102? no:
		// the only exact match is line 102, reached by wrapping around.
		t.Fatalf("case-sensitive: y=%d", c.Y)
	}
	if c.Search(s, "nomatch", false, false) {
		t.Fatal("unexpected match")
	}
}

func TestCopyWordMotions(t *testing.T) {
	s := src("foo bar.baz  qux", "next")
	c := NewCopyMode(100, 0, 0)
	var stops []int
	for i := 0; i < 5; i++ {
		c.WordNext(s)
		stops = append(stops, c.X+100*(c.Y-100))
	}
	if got := strings.Trim(strings.Join(strings.Fields(strings.Trim(fmtInts(stops), "[]")), ","), ","); got != "4,7,8,13,100" {
		t.Fatalf("w stops = %v", stops)
	}
	c.WordPrev(s)
	if c.X != 13 || c.Y != 100 {
		t.Fatalf("b: x=%d y=%d", c.X, c.Y)
	}
	c.X = 0
	c.WordEnd(s)
	if c.X != 2 {
		t.Fatalf("e: x=%d", c.X)
	}
}

func fmtInts(v []int) string {
	var b strings.Builder
	for _, n := range v {
		b.WriteString(itoa(n) + " ")
	}
	return b.String()
}
