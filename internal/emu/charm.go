package emu

import (
	"io"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
)

// charm adapts github.com/charmbracelet/x/vt.
type charm struct {
	e      *vt.Emulator
	modes  Modes
	title  string
	hidden bool
}

func newCharm(cols, rows int) *charm {
	t := &charm{e: vt.NewEmulator(cols, rows)}
	t.e.SetCallbacks(vt.Callbacks{
		Title:       func(s string) { t.title = s },
		EnableMode:  func(m ansi.Mode) { t.setMode(m, true) },
		DisableMode: func(m ansi.Mode) { t.setMode(m, false) },
	})
	return t
}

func (t *charm) setMode(m ansi.Mode, on bool) {
	switch m {
	case ansi.ModeTextCursorEnable:
		t.hidden = !on
	case ansi.ModeCursorKeys:
		t.modes.AppCursor = on
	case ansi.ModeNumericKeypad:
		t.modes.AppKeypad = on
	case ansi.ModeBracketedPaste:
		t.modes.BracketedPaste = on
	case ansi.ModeFocusEvent:
		t.modes.FocusEvents = on
	case ansi.ModeAltScreen, ansi.ModeAltScreenSaveCursor:
		t.modes.AltScreen = on
	case ansi.ModeMouseX10:
		t.modes.MouseX10 = on
	case ansi.ModeMouseNormal:
		t.modes.MouseNormal = on
	case ansi.ModeMouseButtonEvent:
		t.modes.MouseButton = on
	case ansi.ModeMouseAnyEvent:
		t.modes.MouseAny = on
	case ansi.ModeMouseExtSgr:
		t.modes.MouseSGR = on
	}
}

func (t *charm) Write(p []byte) (int, error) { return t.e.Write(p) }
func (t *charm) Resize(cols, rows int)       { t.e.Resize(cols, rows) }
func (t *charm) Size() (int, int)            { return t.e.Width(), t.e.Height() }
func (t *charm) Modes() Modes                { return t.modes }
func (t *charm) Title() string               { return t.title }
func (t *charm) Replies() io.Reader          { return t.e }
func (t *charm) Close() error                { return t.e.Close() }

func (t *charm) Cursor() (int, int, bool) {
	p := t.e.CursorPosition()
	return p.X, p.Y, !t.hidden
}

func (t *charm) Draw(dst uv.Screen, area uv.Rectangle) { t.DrawFrom(dst, area, 0) }

func (t *charm) HistoryLen() int             { return t.e.ScrollbackLen() }
func (t *charm) Pushed() int                 { return t.e.Scrollback().Pushed() }
func (t *charm) SetScrollbackSize(lines int) { t.e.SetScrollbackSize(lines) }

func (t *charm) Line(y int) (uv.Line, bool) {
	if y < 0 {
		sb := t.e.Scrollback()
		l := sb.Line(sb.Len() + y)
		return l, vt.LineWrapped(l)
	}
	if y >= t.e.Height() {
		return nil, false
	}
	w := t.e.Width()
	l := make(uv.Line, w)
	for x := 0; x < w; x++ {
		if c := t.e.CellAt(x, y); c != nil {
			l[x] = *c
		} else {
			l[x] = uv.EmptyCell
		}
	}
	return l, vt.LineWrapped(l)
}

func (t *charm) DrawFrom(dst uv.Screen, area uv.Rectangle, top int) {
	w, h := t.e.Width(), t.e.Height()
	aw, ah := area.Dx(), area.Dy()
	sb := t.e.Scrollback()
	for row := 0; row < ah; row++ {
		y := top + row
		if y >= h {
			break
		}
		var line uv.Line
		if y < 0 {
			line = sb.Line(sb.Len() + y)
		}
		for x := 0; x < w && x < aw; {
			var c *uv.Cell
			if y < 0 {
				c = line.At(x)
			} else {
				c = t.e.CellAt(x, y)
			}
			if c == nil {
				dst.SetCell(area.Min.X+x, area.Min.Y+row, &uv.EmptyCell)
				x++
				continue
			}
			c = vt.StripWrap(c)
			step := c.Width
			if step < 1 {
				step = 1
			}
			if x+step > aw { // wide glyph would straddle the pane edge
				blank := uv.EmptyCell
				blank.Style = c.Style
				dst.SetCell(area.Min.X+x, area.Min.Y+row, &blank)
				break
			}
			dst.SetCell(area.Min.X+x, area.Min.Y+row, c)
			x += step
		}
	}
}
