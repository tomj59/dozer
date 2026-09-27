// Package host owns the real terminal dozer runs in: raw mode, the
// alternate screen, size, drawing, and mirroring the focused pane's
// key/mouse modes.
package host

import (
	"bytes"
	"encoding/base64"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"

	"github.com/tomj59/dozer/internal/emu"
)

// Terminal is the host terminal.
//
// Drawing: callers paint cells into Scr, then call Present. dozer drives
// uv.TerminalRenderer directly rather than uv.TerminalScreen, because
// TerminalScreen.Flush queues the cursor move *after* emitting the frame, so
// the host cursor lagged one frame behind the pane's cursor. That showed up
// as line-editing in the shell drifting off by a keystroke.
type Terminal struct {
	In  *os.File
	Out *os.File
	Scr uv.ScreenBuffer // the frame being composed

	rend        *uv.TerminalRenderer
	frame       bytes.Buffer // renderer output for the current frame
	cursorShown bool

	mu       sync.Mutex
	oldState *term.State
	applied  emu.Modes // modes currently set on the host
	restored bool
}

// Open puts the terminal in raw mode and enters the alternate screen.
// in is the keyboard (os.Stdin, or /dev/tty when stdin was piped).
func Open(in *os.File) (*Terminal, error) {
	t := &Terminal{In: in, Out: os.Stdout}
	st, err := term.MakeRaw(int(t.In.Fd()))
	if err != nil {
		return nil, err
	}
	t.oldState = st

	env := os.Environ()
	t.rend = uv.NewTerminalRenderer(&t.frame, env)
	// The renderer writes to a buffer, so detect colors from the real tty.
	t.rend.SetColorProfile(colorprofile.Detect(t.Out, env))
	t.rend.SetMapNewline(false) // raw mode: no NL→CRNL translation

	w, h := t.Size()
	t.Scr = uv.NewScreenBuffer(w, h)
	t.rend.Resize(w, h)
	t.rend.EnterAltScreen()
	_ = t.rend.Flush()
	_, _ = t.Out.Write(append(t.frame.Bytes(), ansi.HideCursor...))
	t.frame.Reset()
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

// Resize resizes the frame and forces a full repaint on the next Present.
func (t *Terminal) Resize(w, h int) {
	t.Scr.Resize(w, h)
	t.Scr.Touched = nil
	t.rend.Resize(w, h)
	t.rend.Erase()
}

// Redraw forces a full repaint on the next Present.
func (t *Terminal) Redraw() { t.rend.Erase() }

// Present writes the changes in Scr to the terminal, then places the cursor
// at (x, y) and shows it if visible. The cursor is always positioned in the
// same write as the frame, so it never lags.
func (t *Terminal) Present(x, y int, visible bool) error {
	t.rend.Render(t.Scr.RenderBuffer)
	if visible {
		t.rend.MoveTo(x, y)
	}
	_ = t.rend.Flush()
	body := t.frame.Bytes()
	defer t.frame.Reset()

	if len(body) == 0 && visible == t.cursorShown {
		return nil
	}
	var out bytes.Buffer
	if len(body) > 0 {
		out.WriteString(ansi.HideCursor) // no cursor flicker while drawing
		out.Write(body)
	}
	switch {
	case visible:
		out.WriteString(ansi.ShowCursor)
	case t.cursorShown && len(body) == 0:
		out.WriteString(ansi.HideCursor)
	}
	t.cursorShown = visible
	_, err := t.Out.Write(out.Bytes())
	return err
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

// SetClipboard puts text on the system clipboard with OSC 52.
func (t *Terminal) SetClipboard(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, _ = t.Out.WriteString("\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07")
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
	t.frame.Reset()
	t.rend.ExitAltScreen()
	_ = t.rend.Flush()
	_, _ = t.Out.Write(t.frame.Bytes())
	_, _ = t.Out.WriteString("\x1b[0m" + ansi.ShowCursor)
	if t.oldState != nil {
		_ = term.Restore(int(t.In.Fd()), t.oldState)
	}
}
