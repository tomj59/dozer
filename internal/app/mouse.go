package app

import (
	"bytes"

	"github.com/tomj59/dozer/internal/emu"
	"github.com/tomj59/dozer/internal/input"
	"github.com/tomj59/dozer/internal/layout"
	"github.com/tomj59/dozer/internal/pane"
)

// Mouse handling (docs/SPEC.md §4.5, DP-8). dozer owns the host mouse:
//   - click focuses a pane; dragging a divider or title row resizes;
//   - the wheel scrolls a pane's history (copy mode) or, in a full-screen
//     app that doesn't use the mouse, sends arrow keys;
//   - dragging selects text inside one pane only, and copies it (OSC 52);
//   - if the program in a pane turned on mouse reporting (vim, htop), its
//     events go to it instead. Shift+drag still selects with dozer. The
//     host terminal's own selection stays one modifier away (Option in
//     Terminal.app and iTerm2, Shift in most others).

type dragKind int

const (
	dragNone dragKind = iota
	dragBorder
	dragSelect
	dragForward
)

type drag struct {
	kind   dragKind
	pane   int
	dir    layout.Dir // dragBorder
	before bool       // dragBorder: the border before the pane (left/top)
	last   int        // dragBorder: last pointer position along dir
	x, y   int        // dragSelect: where the press was (pane content cells)
	moved  bool       // dragSelect: the pointer moved while pressed
}

// wheelLines is how far one wheel notch scrolls.
const wheelLines = 3

// hit is what's under a canvas position.
type hit struct {
	pane    int // -1: nothing
	title   bool
	divider bool
	x, y    int // content-relative cell (content hits)
}

func (a *App) hitTest(cx, cy int) hit {
	for i, s := range a.visibleSlots() {
		if cx < s.X || cx >= s.X+s.W || cy < s.Y || cy >= s.Y+s.H {
			continue
		}
		if cy == s.Y {
			return hit{pane: i, title: true}
		}
		c := s.Content()
		return hit{pane: i, x: cx - c.X, y: cy - c.Y}
	}
	if !a.zoomed {
		for _, d := range a.res.Dividers {
			if cx != d.X || cy < d.Y || cy >= d.Y+d.H {
				continue
			}
			for i, s := range a.res.Slots { // the pane to the right of it
				if s.X == d.X+1 && cy >= s.Y && cy < s.Y+s.H {
					return hit{pane: i, divider: true}
				}
			}
		}
	}
	return hit{pane: -1}
}

// toCanvas converts a screen cell to canvas coordinates; ok is false on
// the status bar.
func (a *App) toCanvas(x, y int) (int, int, bool) {
	w, h := a.term.Size()
	y0, ah := a.area(w, h)
	// Coordinates are returned even off the canvas: a drag that wanders
	// onto the status bar still needs to know where the pointer is.
	return x + a.view.vx, y - y0 + a.view.vy, y >= y0 && y < y0+ah
}

// repeated is a mouse event and how many times in a row it happened.
type repeated struct {
	ev input.MouseEvent
	n  int
}

// coalesceWheel merges runs of identical wheel events (same direction and
// cell) from one read into a single event with a count, so a fast scroll is
// handled in one step instead of hundreds.
func coalesceWheel(evs []input.MouseEvent) []repeated {
	var out []repeated
	for _, e := range evs {
		if k := len(out) - 1; k >= 0 && e.Wheel() && out[k].ev == e {
			out[k].n++
			continue
		}
		out = append(out, repeated{e, 1})
	}
	return out
}

// mouseN handles an event that happened n times in a row (main loop).
func (a *App) mouseN(e input.MouseEvent, n int) {
	if e.Wheel() {
		a.wheelN = n
		defer func() { a.wheelN = 1 }()
	} else {
		n = 1
	}
	a.mouse(e)
}

// mouse handles one host mouse event (main loop).
func (a *App) mouse(e input.MouseEvent) {
	cx, cy, onCanvas := a.toCanvas(e.X, e.Y)
	d := &a.drag

	// A drag in progress owns every event until the button is released.
	if d.kind != dragNone && !e.Wheel() && (e.Motion || e.Release) {
		a.continueDrag(e, cx, cy)
		if e.Release {
			a.drag = drag{}
		}
		return
	}
	if !onCanvas {
		return
	}
	h := a.hitTest(cx, cy)
	if h.pane < 0 {
		return
	}
	p := a.panes[h.pane]
	modes := p.Modes()

	switch {
	case e.Wheel():
		if !h.title && !h.divider {
			a.wheel(h, p, modes, e)
		}

	case e.Motion || e.Release:
		// Hover, or a release with no drag of ours: programs that track
		// every motion get it.
		if !h.title && !h.divider && wants(modes, e) {
			p.TryInput(e.Encode(h.x, h.y, modes.MouseSGR))
		}

	case e.Button == input.MouseLeft && h.divider:
		*d = drag{kind: dragBorder, pane: h.pane, dir: layout.Cols, before: true, last: cx}

	case e.Button == input.MouseLeft && h.title:
		a.setFocus(h.pane)
		if s := a.full.Slots[h.pane]; s.Y > 0 && !a.zoomed {
			*d = drag{kind: dragBorder, pane: h.pane, dir: layout.Rows, before: true, last: cy}
		}

	default: // a press in a pane's content
		a.setFocus(h.pane)
		if wants(modes, e) && !e.Shift && !p.InCopy() {
			p.TryInput(e.Encode(h.x, h.y, modes.MouseSGR))
			*d = drag{kind: dragForward, pane: h.pane}
			return
		}
		if e.Button != input.MouseLeft {
			return
		}
		*d = drag{kind: dragSelect, pane: h.pane, x: h.x, y: h.y}
		// A click in copy mode moves the cursor there and drops the
		// selection; the selection itself starts when the pointer moves.
		p.WithCopy(func(c *pane.CopyMode, src pane.Source) bool {
			c.Sel = pane.SelNone
			c.X, c.Y = h.x, c.Top+h.y
			c.Move(src, 0, 0)
			return false
		})
	}
}

// continueDrag handles motion and release during a drag.
func (a *App) continueDrag(e input.MouseEvent, cx, cy int) {
	d := &a.drag
	p := a.panes[d.pane]
	switch d.kind {
	case dragBorder:
		pos := cx
		if d.dir == layout.Rows {
			pos = cy
		}
		if delta := pos - d.last; delta != 0 {
			if layout.DragBorder(a.cfg.Layout, d.pane, d.dir, d.before, delta, a.full, a.cfg.MinPane) {
				a.relayout()
			}
			d.last = pos
		}

	case dragForward:
		modes := p.Modes()
		if wants(modes, e) {
			x, y := a.paneCell(d.pane, cx, cy)
			p.TryInput(e.Encode(x, y, modes.MouseSGR))
		}

	case dragSelect:
		x, y := a.paneCell(d.pane, cx, cy)
		if e.Release {
			if !d.moved {
				return // a plain click
			}
			var text string
			p.WithCopy(func(c *pane.CopyMode, src pane.Source) bool {
				text = c.Text(src)
				c.Sel = pane.SelNone
				return c.Transient
			})
			if text != "" {
				a.clipboard(text)
			}
			return
		}
		if !d.moved {
			if x == d.x && y == d.y {
				return
			}
			d.moved = true
			if !p.EnterCopyTransient() {
				a.say("this emulator has no scrollback")
				return
			}
			p.WithCopy(func(c *pane.CopyMode, src pane.Source) bool {
				c.X, c.Y = d.x, c.Top+d.y
				c.Move(src, 0, 0)
				c.Sel = pane.SelNone
				c.Select(pane.SelChar)
				return false
			})
		}
		_, rows := a.paneSize(d.pane)
		p.WithCopy(func(c *pane.CopyMode, src pane.Source) bool {
			// Past the top or bottom edge: scroll, like any terminal.
			switch {
			case y < 0 || cy < a.slotContent(d.pane).Y:
				c.Scroll(src, -1)
			case y >= rows:
				c.Scroll(src, 1)
			}
			c.X, c.Y = x, c.Top+min(max(y, 0), rows-1)
			c.Move(src, 0, 0)
			return false
		})
	}
}

// wheel scrolls a pane: its history, or its program.
func (a *App) wheel(h hit, p *pane.Pane, modes emu.Modes, e input.MouseEvent) {
	up := e.Button == input.MouseWheelUp
	n := max(a.wheelN, 1) // notches (coalesced)
	lines := n * wheelLines
	switch {
	case p.InCopy():
		p.WithCopy(func(c *pane.CopyMode, src pane.Source) bool {
			if up {
				c.Scroll(src, -lines)
			} else {
				c.Scroll(src, lines)
			}
			// Scrolled back to the live screen with nothing selected: done.
			return c.Transient && c.Scrolled(src) == 0 && c.Sel == pane.SelNone
		})
	case modes.WantsMouse() && !e.Shift:
		p.TryInput(bytes.Repeat(e.Encode(h.x, h.y, modes.MouseSGR), n))
	case modes.AltScreen:
		// A full-screen app without mouse support (less, man): arrows,
		// as terminals do ("alternate scroll").
		key := "\x1b[B"
		if up {
			key = "\x1b[A"
		}
		if modes.AppCursor {
			key = "\x1bO" + key[2:]
		}
		p.TryInput(bytes.Repeat([]byte(key), lines))
	case up:
		if p.EnterCopyTransient() {
			p.WithCopy(func(c *pane.CopyMode, src pane.Source) bool {
				c.Scroll(src, -lines)
				return c.Scrolled(src) == 0 // nothing to scroll back to
			})
		}
	}
}

// wants reports whether a program with modes m receives event e.
func wants(m emu.Modes, e input.MouseEvent) bool {
	switch {
	case e.Motion && e.Button == input.MouseNone:
		return m.MouseAny
	case e.Motion:
		return m.MouseButton || m.MouseAny
	case e.Release:
		return m.MouseNormal || m.MouseButton || m.MouseAny
	default:
		return m.WantsMouse()
	}
}

// slotContent is pane i's content rectangle on the canvas.
func (a *App) slotContent(i int) layout.Rect {
	if s, ok := a.visibleSlots()[i]; ok {
		return s.Content()
	}
	return layout.Rect{}
}

func (a *App) paneSize(i int) (int, int) {
	c := a.slotContent(i)
	return c.W, c.H
}

// paneCell converts a canvas position to pane i's content cell, clamped
// to the pane: a drag never leaves the pane it started in.
func (a *App) paneCell(i, cx, cy int) (int, int) {
	c := a.slotContent(i)
	return min(max(cx-c.X, 0), max(c.W-1, 0)), cy - c.Y
}
