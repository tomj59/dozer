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

func (t *charm) Draw(dst uv.Screen, area uv.Rectangle) {
	w, h := t.e.Width(), t.e.Height()
	aw, ah := area.Dx(), area.Dy()
	for y := 0; y < h && y < ah; y++ {
		for x := 0; x < w && x < aw; {
			c := t.e.CellAt(x, y)
			if c == nil {
				dst.SetCell(area.Min.X+x, area.Min.Y+y, &uv.EmptyCell)
				x++
				continue
			}
			step := c.Width
			if step < 1 {
				step = 1
			}
			if x+step > aw { // wide glyph would straddle the pane edge
				blank := uv.EmptyCell
				blank.Style = c.Style
				dst.SetCell(area.Min.X+x, area.Min.Y+y, &blank)
				break
			}
			dst.SetCell(area.Min.X+x, area.Min.Y+y, c)
			x += step
		}
	}
}
