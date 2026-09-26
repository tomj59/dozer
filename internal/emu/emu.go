// Package emu defines the terminal-emulator adapter used by every pane.
//
// Each pane feeds its child process's output into an Emulator, which keeps
// the pane's screen grid. The compositor then asks the Emulator to draw that
// grid into a region of the host screen.
//
// The adapter exists so the emulator library can be swapped (DP-2 in
// docs/SPEC.md). Implementations are NOT safe for concurrent use; the owning
// pane serializes access.
package emu

import (
	"fmt"
	"io"

	uv "github.com/charmbracelet/ultraviolet"
)

// Modes are the terminal modes a program inside a pane has requested.
// dozer mirrors the focused pane's modes onto the host terminal so that raw
// keyboard input arrives encoded the way that program expects.
type Modes struct {
	AppCursor      bool // DECCKM  ?1
	AppKeypad      bool // DECKPAM ESC =
	BracketedPaste bool // ?2004
	FocusEvents    bool // ?1004
	AltScreen      bool // ?1049 / ?1047 / ?47

	MouseX10    bool // ?9
	MouseNormal bool // ?1000
	MouseButton bool // ?1002
	MouseAny    bool // ?1003
	MouseSGR    bool // ?1006
}

// WantsMouse reports whether the program asked for any mouse tracking.
func (m Modes) WantsMouse() bool {
	return m.MouseX10 || m.MouseNormal || m.MouseButton || m.MouseAny
}

// Emulator is a virtual terminal that turns a child's output into a screen.
type Emulator interface {
	// Write feeds output bytes from the child process.
	io.Writer

	// Resize changes the emulated screen size.
	Resize(cols, rows int)

	// Size returns the emulated screen size.
	Size() (cols, rows int)

	// Draw paints the current screen into area of dst. Cells with no
	// explicit colors keep the host terminal's default colors.
	Draw(dst uv.Screen, area uv.Rectangle)

	// Cursor returns the cursor position (relative to the pane) and whether
	// the program wants it shown.
	Cursor() (x, y int, visible bool)

	// Modes returns the currently requested terminal modes.
	Modes() Modes

	// Title returns the last OSC 0/2 window title, if any.
	Title() string

	// Replies returns a reader of bytes the emulator needs sent back to the
	// child (answers to device-attribute and cursor-position queries). It
	// MUST be drained continuously, or Write may block.
	Replies() io.Reader

	// Close releases resources and ends the Replies stream.
	Close() error
}

// Names lists the available emulator back ends.
var Names = []string{"charm", "vt10x"}

// New creates an emulator by back-end name.
func New(name string, cols, rows int) (Emulator, error) {
	switch name {
	case "", "charm":
		return newCharm(cols, rows), nil
	case "vt10x":
		return newVT10x(cols, rows), nil
	}
	return nil, fmt.Errorf("unknown emulator %q (have %v)", name, Names)
}

// MustNew is New for back-end names already validated; it falls back to the
// default back end rather than failing.
func MustNew(name string, cols, rows int) Emulator {
	e, err := New(name, cols, rows)
	if err != nil {
		e = newCharm(cols, rows)
	}
	return e
}
