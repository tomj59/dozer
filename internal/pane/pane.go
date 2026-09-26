// Package pane runs one child process on a pseudo-terminal and keeps its
// screen in an emulator.
package pane

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/creack/pty"

	"github.com/tomj59/dozer/internal/emu"
)

// Spec describes what a pane launches. See docs/SPEC.md §4.3.
type Spec struct {
	Shell string   // interactive shell; default $SHELL, then /bin/sh
	Run   string   // run: command typed into an interactive shell (DP-4)
	Exec  string   // exec: the pane process is `$SHELL -lc Exec`
	Dir   string   // working directory; "" = inherit
	Env   []string // extra KEY=VALUE entries
}

// Argv returns the command line for the spec.
func (s Spec) Argv() []string {
	sh := s.Shell
	if sh == "" {
		sh = os.Getenv("SHELL")
	}
	if sh == "" {
		sh = "/bin/sh"
	}
	switch {
	case s.Exec != "":
		return []string{sh, "-lc", s.Exec}
	case s.Run != "":
		// DP-4: run the command in an interactive shell, then stay at a prompt.
		return []string{sh, "-ic", s.Run + "; exec " + sh}
	default:
		return []string{sh, "-l"}
	}
}

// Pane is a running child process with its emulated screen.
type Pane struct {
	ID int

	mu   sync.Mutex // guards em
	em   emu.Emulator
	ptmx *os.File
	cmd  *exec.Cmd

	dirty func()
	done  chan struct{}
	err   error // exit error, valid after done is closed
}

// Start launches spec on a new PTY of the given size. dirty is called (from
// a background goroutine) whenever the pane's screen changes.
func Start(id int, spec Spec, em emu.Emulator, cols, rows int, dirty func()) (*Pane, error) {
	argv := spec.Argv()
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = spec.Dir
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"DOZER=1",
		"DOZER_PANE="+strconv.Itoa(id),
	)
	cmd.Env = append(cmd.Env, spec.Env...)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	p := &Pane{ID: id, em: em, ptmx: ptmx, cmd: cmd, dirty: dirty, done: make(chan struct{})}

	// Emulator replies (DA, DSR, ...) go back to the child.
	go func() { _, _ = io.Copy(ptmx, em.Replies()) }()
	go p.readLoop()
	return p, nil
}

func (p *Pane) readLoop() {
	buf := make([]byte, 64*1024)
	for {
		n, err := p.ptmx.Read(buf)
		if n > 0 {
			p.mu.Lock()
			_, _ = p.em.Write(buf[:n])
			p.mu.Unlock()
			p.dirty()
		}
		if err != nil {
			break // EIO on Linux / EOF on macOS once the child side closes
		}
	}
	p.err = p.cmd.Wait()
	close(p.done)
	p.dirty()
}

// Input sends bytes to the child as if typed.
func (p *Pane) Input(b []byte) {
	_, _ = p.ptmx.Write(b)
}

// Resize changes both the PTY (the child gets SIGWINCH) and the emulator.
func (p *Pane) Resize(cols, rows int) {
	if cols < 1 || rows < 1 {
		return
	}
	p.mu.Lock()
	p.em.Resize(cols, rows)
	p.mu.Unlock()
	_ = pty.Setsize(p.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// Snapshot is the per-frame state the UI needs besides the cells.
type Snapshot struct {
	CursorX, CursorY int
	CursorVisible    bool
	Modes            emu.Modes
	Title            string
}

// Draw paints the pane into area and returns cursor/mode state.
func (p *Pane) Draw(dst uv.Screen, area uv.Rectangle) Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.em.Draw(dst, area)
	x, y, vis := p.em.Cursor()
	return Snapshot{CursorX: x, CursorY: y, CursorVisible: vis, Modes: p.em.Modes(), Title: p.em.Title()}
}

// Done is closed when the child exits.
func (p *Pane) Done() <-chan struct{} { return p.done }

// ExitCode is the child's exit status (-1 if unknown). Valid after Done.
func (p *Pane) ExitCode() int {
	var ee *exec.ExitError
	if p.err == nil {
		return 0
	}
	if errors.As(p.err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

// Close hangs up the child's process group and releases the PTY.
func (p *Pane) Close() {
	if p.cmd.Process != nil {
		// pty.Start makes the child a session leader, so pgid == pid.
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGHUP)
	}
	_ = p.ptmx.Close()
	p.mu.Lock()
	_ = p.em.Close()
	p.mu.Unlock()
}
