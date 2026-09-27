package vt

// dozer patch: soft-wrap tracking and resize reflow (dozer DP-7).
//
// A line that the cursor ran off the end of (autowrap) is marked "wrapped" by
// a marker on its last cell. Keeping the marker in the cell, rather than in a
// parallel slice, means every existing line operation (scroll, insert/delete
// line, erase, scrollback push) carries or clears it for free. Consumers that
// copy cells out (Draw) must strip it with StripWrap.

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
)

// wrapMark is appended to Link.Params of a wrapped line's last cell. A NUL
// cannot appear in real OSC 8 params, so it can't collide with a hyperlink.
const wrapMark = "\x00dozer-wrap"

func isWrapCell(c *uv.Cell) bool { return c != nil && strings.HasSuffix(c.Link.Params, wrapMark) }

func setWrapCell(c *uv.Cell) {
	if c != nil && !isWrapCell(c) {
		c.Link.Params += wrapMark
	}
}

// StripWrap returns c without the wrap marker (c itself when it has none).
func StripWrap(c *uv.Cell) *uv.Cell {
	if !isWrapCell(c) {
		return c
	}
	cc := *c
	cc.Link.Params = strings.TrimSuffix(cc.Link.Params, wrapMark)
	return &cc
}

// LineWrapped reports whether line continues on the next line (soft wrap).
func LineWrapped(line uv.Line) bool {
	return len(line) > 0 && isWrapCell(&line[len(line)-1])
}

// markWrapped flags screen row y as soft-wrapped.
func (s *Screen) markWrapped(y int) {
	if y < 0 || y >= s.buf.Height() {
		return
	}
	line := s.buf.Line(y)
	if len(line) > 0 {
		setWrapCell(&line[len(line)-1])
	}
}

// Pushed returns how many lines were ever pushed, so callers can keep a
// stable position while old lines are evicted: the absolute index of
// Line(i) is Pushed()-Len()+i.
func (s *Scrollback) Pushed() int {
	if s == nil {
		return 0
	}
	return s.pushed
}

// cellWidth is the number of columns c occupies; 0 for wide-glyph
// continuation cells.
func cellWidth(c *uv.Cell) int {
	if c.Width == 0 && c.Content == "" {
		return 0
	}
	return max(c.Width, 1)
}

// logicalLine is one unwrapped line of text, as the physical rows it was
// spread over (the last row trimmed of trailing blanks).
type logicalLine struct {
	segs   []uv.Line
	cursor int // column offset of the cursor in this line, or -1
}

// reflow rewraps the screen and its scrollback to width×height. cx, cy is the
// cursor (phantom: pending wrap). It returns the new cursor position.
func (s *Screen) reflow(width, height, cx, cy int, phantom bool) (int, int, bool) {
	oldW, oldH := s.buf.Width(), s.buf.Height()
	if width <= 0 || height <= 0 || oldW <= 0 {
		s.Resize(width, height)
		return min(cx, max(width-1, 0)), min(cy, max(height-1, 0)), false
	}

	// Screen rows that matter: everything up to the last non-blank row or
	// the cursor, whichever is lower.
	last := cy
	for y := oldH - 1; y > last; y-- {
		if !s.isLineEmpty(s.buf.Line(y)) {
			last = y
			break
		}
	}

	var logical []logicalLine
	cur := logicalLine{cursor: -1}
	start := 0 // columns in cur so far
	add := func(line uv.Line, isCursorRow bool) {
		wrapped := LineWrapped(line)
		n := len(line)
		if !wrapped {
			for n > 0 && isBlankCell(&line[n-1]) {
				n--
			}
		}
		cur.segs = append(cur.segs, line[:n:n])
		if isCursorRow {
			off := cx
			if phantom {
				off++
			}
			cur.cursor = start + off
		}
		start += n
		if !wrapped {
			logical = append(logical, cur)
			cur, start = logicalLine{cursor: -1}, 0
		}
	}
	if s.scrollback != nil {
		for _, l := range s.scrollback.lines {
			add(l, false)
		}
	}
	for y := 0; y <= last && y < oldH; y++ {
		add(s.buf.Line(y), y == cy)
	}
	if len(cur.segs) > 0 {
		logical = append(logical, cur)
	}

	// Rewrap. Rows hold only their content (no padding) to keep this cheap
	// with a full scrollback; screen rows are padded to width at the end.
	// A line that still fits in one row reuses its cells without copying.
	var rows []uv.Line
	var wrapped []bool
	curRow, curCol, curPhantom := 0, 0, false
	for _, ll := range logical {
		if len(ll.segs) == 1 && len(ll.segs[0]) <= width && ll.cursor <= width {
			if ll.cursor >= 0 {
				curRow, curCol = len(rows), ll.cursor
			}
			rows, wrapped = append(rows, ll.segs[0]), append(wrapped, false)
			continue
		}
		row := make(uv.Line, 0, width)
		x, col := 0, 0 // x: column in row; col: logical column
		for _, seg := range ll.segs {
			for i := range seg {
				c := seg[i]
				w := cellWidth(&c)
				if w == 0 { // continuation; regenerated below
					col++
					continue
				}
				if w > width { // can't fit at all; drop
					col++
					continue
				}
				if x+w > width {
					rows, wrapped = append(rows, row), append(wrapped, true)
					row, x = make(uv.Line, 0, width), 0
				}
				if ll.cursor == col {
					curRow, curCol = len(rows), x
				}
				row = append(row, *StripWrap(&c))
				for k := 1; k < w; k++ {
					row = append(row, uv.Cell{})
				}
				x += w
				col++
			}
		}
		if ll.cursor >= col {
			// Cursor at or past the end of the text (e.g. after a prompt's
			// trailing space, which was trimmed).
			x += ll.cursor - col
			for x > width {
				rows, wrapped = append(rows, row), append(wrapped, true)
				row = nil
				x -= width
			}
			curRow, curCol = len(rows), x
		}
		rows, wrapped = append(rows, row), append(wrapped, false)
	}
	if curCol >= width {
		curCol, curPhantom = width-1, true
	}

	// The last height rows are the screen; anything above goes to scrollback.
	top := max(0, len(rows)-height)
	if curRow < top { // keep the cursor visible
		top = curRow
	}
	if s.scrollback != nil {
		s.scrollback.Clear()
		for i, l := range rows[:top] {
			if wrapped[i] {
				l = padLine(l, width)
				setWrapCell(&l[width-1])
			}
			s.scrollback.pushOwned(l)
		}
	}
	buf := uv.NewRenderBuffer(width, height)
	for y := 0; y < height && top+y < len(rows); y++ {
		l := padLine(rows[top+y], width)
		if wrapped[top+y] {
			setWrapCell(&l[width-1])
		}
		buf.Lines[y] = l
	}
	s.buf = buf
	s.scroll = s.buf.Bounds()
	s.touchArea(s.buf.Bounds())
	return curCol, curRow - top, curPhantom
}

// padLine extends l with blank cells to width columns.
func padLine(l uv.Line, width int) uv.Line {
	for len(l) < width {
		l = append(l, uv.EmptyCell)
	}
	return l
}

// pushOwned appends an already trimmed line that the caller gives up.
func (s *Scrollback) pushOwned(l uv.Line) {
	if len(s.lines) >= s.maxLines {
		s.lines[0] = nil
		s.lines = s.lines[1:]
	}
	s.lines = append(s.lines, l)
	s.pushed++
}
