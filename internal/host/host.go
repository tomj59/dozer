// Package host owns the real terminal dozer runs in: raw mode, the
// alternate screen, size, and mirroring the focused pane's key/mouse modes.
package host

import (
	"os"
	"strings"
	"sync"

	uv "github.com/charmbracelet/ultraviolet"
	"golang.org/x/term"

	"github.com/tomj59/dozer/internal/emu"
)

// Terminal is the host terminal.
type Terminal struct {
	In  *os.File
	Out *os.File
	Scr *uv.TerminalScreen

	mu       sync.Mutex
	oldState *term.State
	applied  emu.Modes // modes currently set on the host
	restored bool
}

// Open puts the terminal in raw mode and enters the alternate screen.
func Open() (*Terminal, error) {
	t := &Terminal{In: os.Stdin, Out: os.Stdout}
	st, err := term.MakeRaw(int(t.In.Fd()))
	if err != nil {
		return nil, err
	}
	t.oldState = st
	t.Scr = uv.NewTerminalScreen(t.Out, os.Environ())
	w, h := t.Size()
	_ = t.Scr.Resize(w, h)
	_ = t.Scr.EnterAltScreen()
	_ = t.Scr.Flush()
	return t, nil
}

// Size returns the host terminal size (80x24 if unknown).
func (t *Terminal) Size() (int, int) {
	w, h, err := term.GetSize(int(t.Out.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		return 80, 24
	}
	return w, h
}

// Mirror sets the host's key, paste, focus and mouse modes to match m, the
// focused pane's modes, emitting only what changed.
func (t *Terminal) Mirror(m emu.Modes) {
	t.mu.Lock()
	defer t.mu.Unlock()
	var b strings.Builder
	dec := func(on, was bool, n string) {
		if on == was {
			return
		}
		b.WriteString("\x1b[?" + n)
		if on {
			b.WriteByte('h')
		} else {
			b.WriteByte('l')
		}
	}
	a := t.applied
	dec(m.AppCursor, a.AppCursor, "1")
	if m.AppKeypad != a.AppKeypad {
		if m.AppKeypad {
			b.WriteString("\x1b=")
		} else {
			b.WriteString("\x1b>")
		}
	}
	dec(m.BracketedPaste, a.BracketedPaste, "2004")
	dec(m.FocusEvents, a.FocusEvents, "1004")
	dec(m.MouseX10, a.MouseX10, "9")
	dec(m.MouseNormal, a.MouseNormal, "1000")
	dec(m.MouseButton, a.MouseButton, "1002")
	dec(m.MouseAny, a.MouseAny, "1003")
	dec(m.MouseSGR, a.MouseSGR, "1006")
	if b.Len() > 0 {
		_, _ = t.Out.WriteString(b.String())
		t.applied = m
	}
}

// Restore undoes everything Open and Mirror did. Safe to call twice.
func (t *Terminal) Restore() {
	t.Mirror(emu.Modes{}) // turn off every mirrored mode
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.restored {
		return
	}
	t.restored = true
	_ = t.Scr.ExitAltScreen()
	_ = t.Scr.ShowCursor()
	_ = t.Scr.Flush()
	_, _ = t.Out.WriteString("\x1b[0m\x1b[?25h")
	if t.oldState != nil {
		_ = term.Restore(int(t.In.Fd()), t.oldState)
	}
}
