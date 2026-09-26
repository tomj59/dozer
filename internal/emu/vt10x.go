package emu

import (
	"image/color"
	"io"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/hinshun/vt10x"
)

// vt10xEmu adapts github.com/hinshun/vt10x. It is kept only as a DP-2
// comparison back end: no wide characters, no scrollback, no bracketed-paste
// tracking, and 24-bit colors collide with the 256-color range.
type vt10xEmu struct {
	t  vt10x.Terminal
	pr *io.PipeReader
	pw *io.PipeWriter
}

// vt10x glyph attribute bits (unexported upstream).
const (
	vtReverse = 1 << iota
	vtUnderline
	vtBold
	vtGfx
	vtItalic
	vtBlink
)

func newVT10x(cols, rows int) *vt10xEmu {
	pr, pw := io.Pipe()
	return &vt10xEmu{
		t:  vt10x.New(vt10x.WithWriter(pw), vt10x.WithSize(cols, rows)),
		pr: pr,
		pw: pw,
	}
}

func (v *vt10xEmu) Write(p []byte) (int, error) { return v.t.Write(p) }
func (v *vt10xEmu) Resize(cols, rows int)       { v.t.Resize(cols, rows) }
func (v *vt10xEmu) Size() (int, int)            { return v.t.Size() }
func (v *vt10xEmu) Title() string               { return v.t.Title() }
func (v *vt10xEmu) Replies() io.Reader          { return v.pr }
func (v *vt10xEmu) Close() error                { return v.pw.Close() }

func (v *vt10xEmu) Cursor() (int, int, bool) {
	c := v.t.Cursor()
	return c.X, c.Y, v.t.CursorVisible()
}

func (v *vt10xEmu) Modes() Modes {
	m := v.t.Mode()
	return Modes{
		AppCursor:   m&vt10x.ModeAppCursor != 0,
		AppKeypad:   m&vt10x.ModeAppKeypad != 0,
		FocusEvents: m&vt10x.ModeFocus != 0,
		AltScreen:   m&vt10x.ModeAltScreen != 0,
		MouseX10:    m&vt10x.ModeMouseX10 != 0,
		MouseNormal: m&vt10x.ModeMouseButton != 0,
		MouseButton: m&vt10x.ModeMouseMotion != 0,
		MouseAny:    m&vt10x.ModeMouseMany != 0,
		MouseSGR:    m&vt10x.ModeMouseSgr != 0,
	}
}

func vtColor(c vt10x.Color) color.Color {
	switch {
	case c >= vt10x.DefaultFG:
		return nil // host default
	case c < 16:
		return ansi.BasicColor(c)
	case c < 256:
		return ansi.IndexedColor(c)
	default:
		return ansi.TrueColor(uint32(c))
	}
}

func (v *vt10xEmu) Draw(dst uv.Screen, area uv.Rectangle) {
	v.t.Lock()
	defer v.t.Unlock()
	cols, rows := v.t.Size()
	for y := 0; y < rows && y < area.Dy(); y++ {
		for x := 0; x < cols && x < area.Dx(); x++ {
			g := v.t.Cell(x, y)
			ch := g.Char
			if ch == 0 {
				ch = ' '
			}
			cell := uv.Cell{Content: string(ch), Width: 1}
			cell.Style.Fg = vtColor(g.FG)
			cell.Style.Bg = vtColor(g.BG)
			if g.Mode&vtReverse != 0 {
				cell.Style.Attrs |= uv.AttrReverse
			}
			if g.Mode&vtBold != 0 {
				cell.Style.Attrs |= uv.AttrBold
			}
			if g.Mode&vtItalic != 0 {
				cell.Style.Attrs |= uv.AttrItalic
			}
			if g.Mode&vtBlink != 0 {
				cell.Style.Attrs |= uv.AttrBlink
			}
			if g.Mode&vtUnderline != 0 {
				cell.Style.Underline = uv.UnderlineSingle
			}
			dst.SetCell(area.Min.X+x, area.Min.Y+y, &cell)
		}
	}
}
