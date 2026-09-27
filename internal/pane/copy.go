package pane

import (
	"strings"
	"unicode"

	uv "github.com/charmbracelet/ultraviolet"
)

// Source is the text a CopyMode moves over: a pane's scrollback plus screen,
// addressed by absolute line numbers that don't change as output arrives.
type Source interface {
	// Line returns the cells of absolute line n and whether it soft-wraps.
	Line(n int) (uv.Line, bool)
	// First and Last are the oldest and newest absolute line numbers.
	First() int
	Last() int
	// Rows and Cols are the pane's size.
	Rows() int
	Cols() int
}

// CopyMode is a pane's scrolled-back view with a keyboard cursor and an
// optional selection (tmux's copy mode, vi keys). It is plain state: the
// owner serializes access and supplies the Source on every call.
type CopyMode struct {
	Top    int // absolute line shown on the view's first row
	X, Y   int // cursor (column, absolute line)
	Sel    SelMode
	AX, AY int // selection anchor

	// Mouse selection started a drag in this mode; the view exits when the
	// selection is copied (entered by the mouse, left by the mouse).
	Transient bool

	search    string
	backwards bool
	wantX     int // column to return to on vertical moves (like vi)
}

// SelMode is the kind of selection.
type SelMode int

const (
	SelNone SelMode = iota
	SelChar         // v: from anchor to cursor, following the text
	SelLine         // V: whole lines
)

// NewCopyMode starts copy mode on the live screen with the cursor at x, y
// (screen coordinates); top is the absolute number of screen row 0.
func NewCopyMode(top, x, y int) *CopyMode {
	return &CopyMode{Top: top, X: x, Y: top + y, wantX: x}
}

// Scrolled reports how many lines the view is above the live screen.
func (c *CopyMode) Scrolled(src Source) int { return liveTop(src) - c.Top }

func liveTop(src Source) int { return src.Last() - src.Rows() + 1 }

// clamp keeps the view and cursor inside the text and the cursor on screen.
func (c *CopyMode) clamp(src Source) {
	first, last := src.First(), src.Last()
	live := liveTop(src)
	c.Y = min(max(c.Y, first), last)
	c.X = min(max(c.X, 0), max(src.Cols()-1, 0))
	c.Top = min(max(c.Top, first), live)
	if c.Y < c.Top {
		c.Top = c.Y
	}
	if c.Y >= c.Top+src.Rows() {
		c.Top = c.Y - src.Rows() + 1
	}
	c.Top = min(max(c.Top, first), live)
}

// Scroll moves the view by n lines (negative = back in history). The cursor
// stays on the same screen row.
func (c *CopyMode) Scroll(src Source, n int) {
	row := c.Y - c.Top
	c.Top += n
	c.Top = min(max(c.Top, src.First()), liveTop(src))
	c.Y = c.Top + row
	c.clamp(src)
}

// Move moves the cursor by dx, dy.
func (c *CopyMode) Move(src Source, dx, dy int) {
	if dx != 0 {
		c.X += dx
		c.wantX = c.X
	}
	if dy != 0 {
		c.Y += dy
		c.X = c.wantX
	}
	c.clamp(src)
}

// Page moves the cursor and view by a fraction of the pane height (n
// halves: -2 = page up, 1 = half page down, …).
func (c *CopyMode) Page(src Source, halves int) {
	d := halves * max(src.Rows()/2, 1)
	c.Top += d
	c.Y += d
	c.clamp(src)
}

// Goto moves to an absolute line (first/last of history) keeping x.
func (c *CopyMode) Goto(src Source, line int) {
	c.Y = line
	c.X = c.wantX
	c.clamp(src)
}

// ScreenRow puts the cursor on a view row: 0 top, -1 bottom, rows/2 middle.
func (c *CopyMode) ScreenRow(src Source, row int) {
	if row < 0 {
		row = src.Rows() + row
	}
	c.Y = c.Top + row
	c.clamp(src)
}

// LineStart / LineEnd / FirstNonBlank move within the cursor line.
func (c *CopyMode) LineStart(src Source) { c.X, c.wantX = 0, 0 }

func (c *CopyMode) LineEnd(src Source) {
	l, _ := src.Line(c.Y)
	c.X = max(textWidth(l)-1, 0)
	c.wantX = c.X
	c.clamp(src)
}

func (c *CopyMode) FirstNonBlank(src Source) {
	l, _ := src.Line(c.Y)
	c.X = 0
	for i := range l {
		if !isSpaceCell(&l[i]) {
			c.X = i
			break
		}
	}
	c.wantX = c.X
}

// cls classifies a cell for word motions: 0 space, 1 punctuation, 2 word.
func cls(l uv.Line, x int) int {
	if x < 0 || x >= len(l) || isSpaceCell(&l[x]) {
		return 0
	}
	r := []rune(l[x].Content)
	if len(r) > 0 && (unicode.IsLetter(r[0]) || unicode.IsDigit(r[0]) || r[0] == '_') {
		return 2
	}
	if l[x].Width == 0 && l[x].Content == "" {
		return 2 // continuation of a wide glyph
	}
	return 1
}

// WordNext moves to the start of the next word (vi w), across lines.
func (c *CopyMode) WordNext(src Source) {
	l, _ := src.Line(c.Y)
	x, y := c.X, c.Y
	k := cls(l, x)
	for x < len(l) && k != 0 && cls(l, x) == k {
		x++
	}
	for {
		for x < len(l) && cls(l, x) == 0 {
			x++
		}
		if x < len(l) || y >= src.Last() {
			break
		}
		y++
		l, _ = src.Line(y)
		x = 0
	}
	c.X, c.Y, c.wantX = min(x, max(len(l)-1, 0)), y, x
	c.clamp(src)
}

// WordPrev moves to the start of the previous word (vi b).
func (c *CopyMode) WordPrev(src Source) {
	l, _ := src.Line(c.Y)
	x, y := c.X-1, c.Y
	for {
		for x >= 0 && cls(l, x) == 0 {
			x--
		}
		if x >= 0 || y <= src.First() {
			break
		}
		y--
		l, _ = src.Line(y)
		x = len(l) - 1
	}
	if x < 0 {
		x = 0
	} else {
		k := cls(l, x)
		for x > 0 && cls(l, x-1) == k {
			x--
		}
	}
	c.X, c.Y, c.wantX = x, y, x
	c.clamp(src)
}

// WordEnd moves to the end of the word (vi e).
func (c *CopyMode) WordEnd(src Source) {
	l, _ := src.Line(c.Y)
	x, y := c.X+1, c.Y
	for {
		for x < len(l) && cls(l, x) == 0 {
			x++
		}
		if x < len(l) || y >= src.Last() {
			break
		}
		y++
		l, _ = src.Line(y)
		x = 0
	}
	if x < len(l) {
		k := cls(l, x)
		for x+1 < len(l) && cls(l, x+1) == k {
			x++
		}
	}
	c.X, c.Y, c.wantX = min(x, max(len(l)-1, 0)), y, x
	c.clamp(src)
}

// Select starts (or with the same mode, clears) a selection at the cursor.
func (c *CopyMode) Select(mode SelMode) {
	if c.Sel == mode {
		c.Sel = SelNone
		return
	}
	if c.Sel == SelNone {
		c.AX, c.AY = c.X, c.Y
	}
	c.Sel = mode
}

// span returns the selection's ordered endpoints.
func (c *CopyMode) span() (x1, y1, x2, y2 int) {
	x1, y1, x2, y2 = c.AX, c.AY, c.X, c.Y
	if y2 < y1 || y2 == y1 && x2 < x1 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}
	return
}

// Selected reports whether cell x of absolute line y is in the selection.
func (c *CopyMode) Selected(x, y int) bool {
	if c.Sel == SelNone {
		return false
	}
	x1, y1, x2, y2 := c.span()
	if y < y1 || y > y2 {
		return false
	}
	if c.Sel == SelLine {
		return true
	}
	return (y > y1 || x >= x1) && (y < y2 || x <= x2)
}

// Text returns the selected text. Soft-wrapped lines are joined, so a long
// line copies as one line; trailing blanks are dropped.
func (c *CopyMode) Text(src Source) string {
	if c.Sel == SelNone {
		return ""
	}
	x1, y1, x2, y2 := c.span()
	if c.Sel == SelLine {
		x1, x2 = 0, 1<<30
	}
	var b strings.Builder
	for y := y1; y <= y2; y++ {
		l, wrapped := src.Line(y)
		from, to := 0, len(l)-1
		if y == y1 {
			from = x1
		}
		if y == y2 {
			to = min(x2, len(l)-1)
		}
		var line strings.Builder
		for x := from; x <= to && x < len(l); x++ {
			cell := &l[x]
			if cell.Width == 0 && cell.Content == "" {
				continue // wide-glyph continuation
			}
			if cell.Content == "" {
				line.WriteByte(' ')
			} else {
				line.WriteString(cell.Content)
			}
		}
		s := line.String()
		joined := wrapped && y < y2 && to >= len(l)-1
		if !joined {
			s = strings.TrimRight(s, " ")
		}
		b.WriteString(s)
		if y < y2 && !joined {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// Search finds query from the cursor (backwards: towards history) and moves
// the cursor to the match. An empty query repeats the last search; reverse
// flips its direction (vi n / N). It reports whether anything matched.
func (c *CopyMode) Search(src Source, query string, backwards, reverse bool) bool {
	if query != "" {
		c.search, c.backwards = query, backwards
	}
	if c.search == "" {
		return false
	}
	back := c.backwards != reverse
	q := c.search
	fold := strings.ToLower(q) == q // smart case
	if fold {
		q = strings.ToLower(q)
	}
	first, last := src.First(), src.Last()
	n := last - first + 1
	for i := 0; i <= n; i++ {
		y := c.Y
		if back {
			y -= i
		} else {
			y += i
		}
		y = first + ((y-first)%n+n)%n // wrap around
		l, _ := src.Line(y)
		text, cols := lineText(l)
		if fold {
			text = strings.ToLower(text)
		}
		idx := allIndex(text, q)
		if back {
			for j := len(idx) - 1; j >= 0; j-- {
				x := cols[idx[j]]
				if i > 0 || x < c.X {
					c.X, c.Y, c.wantX = x, y, x
					c.clamp(src)
					return true
				}
			}
		} else {
			for _, bi := range idx {
				x := cols[bi]
				if i > 0 || x > c.X {
					c.X, c.Y, c.wantX = x, y, x
					c.clamp(src)
					return true
				}
			}
		}
	}
	return false
}

// SearchTerm is the active search, for display.
func (c *CopyMode) SearchTerm() string { return c.search }

// lineText returns a line as a string and, for each byte offset where a
// cell starts, the cell's column.
func lineText(l uv.Line) (string, map[int]int) {
	var b strings.Builder
	cols := map[int]int{}
	for x := range l {
		cell := &l[x]
		if cell.Width == 0 && cell.Content == "" {
			continue
		}
		cols[b.Len()] = x
		if cell.Content == "" {
			b.WriteByte(' ')
		} else {
			b.WriteString(cell.Content)
		}
	}
	return b.String(), cols
}

func allIndex(s, sub string) []int {
	var out []int
	for i := 0; ; {
		j := strings.Index(s[i:], sub)
		if j < 0 {
			return out
		}
		out = append(out, i+j)
		i += j + max(len(sub), 1)
		if i > len(s) {
			return out
		}
	}
}

func isSpaceCell(c *uv.Cell) bool {
	return (c.Content == "" || c.Content == " ") && c.Width <= 1 && !(c.Width == 0 && c.Content == "")
}

// textWidth is the number of columns up to the last non-blank cell.
func textWidth(l uv.Line) int {
	n := len(l)
	for n > 0 && isSpaceCell(&l[n-1]) {
		n--
	}
	return n
}
