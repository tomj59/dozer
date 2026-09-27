package pane

import (
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/tomj59/dozer/internal/emu"
)

// emuSource adapts a pane's emulator to Source.
type emuSource struct {
	h          emu.History
	em         emu.Emulator
	cols, rows int
}

func (s emuSource) Line(n int) (uv.Line, bool) {
	if s.h == nil {
		return nil, false
	}
	return s.h.Line(n - s.h.Pushed())
}

func (s emuSource) First() int {
	if s.h == nil {
		return 0
	}
	if s.em.Modes().AltScreen { // a full-screen app's screen has no history
		return s.h.Pushed()
	}
	return s.h.Pushed() - s.h.HistoryLen()
}

func (s emuSource) Last() int {
	if s.h == nil {
		return s.rows - 1
	}
	return s.h.Pushed() + s.rows - 1
}

func (s emuSource) Rows() int { return s.rows }
func (s emuSource) Cols() int { return s.cols }

func (p *Pane) sourceLocked() emuSource {
	h, _ := p.em.(emu.History)
	return emuSource{h: h, em: p.em, cols: p.cols, rows: p.rows}
}

// SetScrollback sets how many lines of history the pane keeps (from the
// next start on; call before Start).
func (p *Pane) SetScrollback(lines int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scrollback = lines
}

// HasHistory reports whether the pane's emulator supports scrollback.
func (p *Pane) HasHistory() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.em.(emu.History)
	return ok
}

// InCopy reports whether the pane shows copy mode (a scrolled-back view).
func (p *Pane) InCopy() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cm != nil
}

// EnterCopy starts copy mode at the live screen with the cursor where the
// program's cursor is. It does nothing if already in copy mode.
func (p *Pane) EnterCopy() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.em == nil {
		return false
	}
	if _, ok := p.em.(emu.History); !ok {
		return false
	}
	if p.cm == nil {
		src := p.sourceLocked()
		x, y, _ := p.em.Cursor()
		p.cm = NewCopyMode(liveTop(src), x, y)
		p.cm.clamp(src)
	}
	return true
}

// ExitCopy returns the pane to its live screen.
func (p *Pane) ExitCopy() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cm = nil
}

// WithCopy runs f on the pane's copy mode (if any) under the pane lock.
// If f returns true, copy mode ends.
func (p *Pane) WithCopy(f func(cm *CopyMode, src Source) (exit bool)) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cm == nil {
		return false
	}
	if f(p.cm, p.sourceLocked()) {
		p.cm = nil
	}
	return true
}

// drawCopyLocked paints the copy-mode view: history from cm.Top, the
// selection in reverse video, and a position indicator in the top right.
func (p *Pane) drawCopyLocked(dst uv.Screen, area uv.Rectangle, src emuSource) {
	top := p.cm.Top - src.h.Pushed()
	src.h.DrawFrom(dst, area, top)
	if p.cm.Sel != SelNone {
		for row := 0; row < area.Dy(); row++ {
			y := p.cm.Top + row
			for x := 0; x < area.Dx(); x++ {
				if !p.cm.Selected(x, y) {
					continue
				}
				c := dst.CellAt(area.Min.X+x, area.Min.Y+row)
				if c == nil {
					continue
				}
				cc := *c
				cc.Style.Attrs ^= uv.AttrReverse
				dst.SetCell(area.Min.X+x, area.Min.Y+row, &cc)
			}
		}
	}
	// Position indicator, like tmux: [lines back/history].
	tag := "[" + itoa(p.cm.Scrolled(src)) + "/" + itoa(src.Last()-src.First()+1-src.Rows()) + "]"
	x0 := area.Max.X - len(tag)
	if x0 < area.Min.X {
		return
	}
	for i, r := range tag {
		c := uv.Cell{Content: string(r), Width: 1}
		c.Style.Attrs = uv.AttrReverse
		dst.SetCell(x0+i, area.Min.Y, &c)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// Modes returns the terminal modes the pane's program asked for (none once
// it has ended).
func (p *Pane) Modes() emu.Modes {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.em == nil || p.state.Dead() {
		return emu.Modes{}
	}
	return p.em.Modes()
}

// EnterCopyTransient starts copy mode for the mouse: it ends by itself once
// a drag is copied or the wheel scrolls back to the bottom.
func (p *Pane) EnterCopyTransient() bool {
	if p.InCopy() {
		return true
	}
	if !p.EnterCopy() {
		return false
	}
	p.WithCopy(func(c *CopyMode, _ Source) bool { c.Transient = true; return false })
	return true
}
