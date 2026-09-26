// Package pane runs one child process on a pseudo-terminal and keeps its
// screen in an emulator. A pane is a permanent slot: when its process
// exits, the pane stays (dead) until it is restarted (docs/SPEC.md §4.10).
package pane

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/creack/pty"
	"golang.org/x/sys/unix"

	"github.com/tomj59/dozer/internal/emu"
)

// Restart policies for automatic restarts.
const (
	RestartNever     = "never"
	RestartOnFailure = "on-failure"
	RestartAlways    = "always"
)

// Spec describes what a pane launches. See docs/SPEC.md §4.3.
type Spec struct {
	Title   string   // label; default derived from the command
	Shell   string   // interactive shell; default $SHELL, then /bin/sh
	Run     string   // run: command in an interactive shell, then a prompt (DP-4)
	Exec    string   // exec: the pane process is `$SHELL -lc Exec`
	Dir     string   // working directory; "" = inherit
	Env     []string // extra KEY=VALUE entries
	Restart string   // never | on-failure | always (automatic restarts)
}

func (s Spec) shell() string {
	if s.Shell != "" {
		return s.Shell
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return "/bin/sh"
}

// Argv returns the command line for the spec.
func (s Spec) Argv() []string {
	sh := s.shell()
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

// Label is the default title: the command, or the shell's name.
func (s Spec) Label() string {
	switch {
	case s.Title != "":
		return s.Title
	case s.Exec != "":
		return s.Exec
	case s.Run != "":
		return s.Run
	}
	return filepath.Base(s.shell())
}

// State is a pane's lifecycle state.
type State int

const (
	Running      State = iota
	Exited             // status 0
	Failed             // non-zero status or killed by a signal
	Disconnected       // heuristic: exec'd ssh exited 255
)

// Dead reports whether the pane's process has ended.
func (s State) Dead() bool { return s != Running }

// Pane is a slot with (maybe) a running process and its emulated screen.
type Pane struct {
	ID   int
	Spec Spec

	emuName string
	dirty   func()

	mu       sync.Mutex // guards everything below
	em       emu.Emulator
	ptmx     *os.File
	cmd      *exec.Cmd
	cols     int
	rows     int
	gen      int // incremented per start; stale exits are ignored
	state    State
	code     int    // exit status (Exited/Failed)
	signal   string // e.g. "SIGKILL" if killed by a signal
	exitedAt time.Time
	restarts int         // consecutive automatic restarts (for back-off)
	timer    *time.Timer // pending automatic restart
}

// New creates a pane; call Start to launch its process. dirty is called
// (from background goroutines) whenever the pane's display changes.
func New(id int, spec Spec, emuName string, cols, rows int, dirty func()) *Pane {
	return &Pane{ID: id, Spec: spec, emuName: emuName, cols: max(cols, 1), rows: max(rows, 1), dirty: dirty}
}

// Start launches (or relaunches) the pane's process on a fresh screen.
func (p *Pane) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.startLocked()
}

func (p *Pane) startLocked() error {
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
	p.stopLocked()
	em, err := emu.New(p.emuName, p.cols, p.rows)
	if err != nil {
		return err
	}
	argv := p.Spec.Argv()
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = p.Spec.Dir
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"DOZER=1",
		"DOZER_PANE="+strconv.Itoa(p.ID),
	)
	cmd.Env = append(cmd.Env, p.Spec.Env...)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(p.cols), Rows: uint16(p.rows)})
	if err != nil {
		_ = em.Close()
		// Show the failure in the pane instead of aborting dozer.
		p.em = emu.MustNew(p.emuName, p.cols, p.rows)
		_, _ = p.em.Write([]byte("dozer: cannot start " + strings.Join(argv, " ") + ": " + err.Error() + "\r\n"))
		p.state, p.code, p.signal, p.exitedAt = Failed, 127, "", time.Now()
		return err
	}
	p.gen++
	p.em, p.ptmx, p.cmd = em, ptmx, cmd
	p.state, p.code, p.signal = Running, 0, ""
	gen := p.gen
	go func() { _, _ = io.Copy(ptmx, em.Replies()) }() // DA/DSR replies to the child
	go p.readLoop(gen, ptmx, em, cmd)
	return nil
}

// stopLocked hangs up the current process (if any) without waiting.
func (p *Pane) stopLocked() {
	if p.cmd != nil && p.cmd.Process != nil && p.state == Running {
		// pty.Start makes the child a session leader, so pgid == pid.
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGHUP)
	}
	if p.ptmx != nil {
		_ = p.ptmx.Close()
		p.ptmx = nil
	}
	if p.em != nil {
		_ = p.em.Close()
	}
	p.gen++ // any exit still in flight belongs to the old process
}

func (p *Pane) readLoop(gen int, ptmx *os.File, em emu.Emulator, cmd *exec.Cmd) {
	buf := make([]byte, 64*1024)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			p.mu.Lock()
			if p.gen == gen {
				_, _ = em.Write(buf[:n])
			}
			p.mu.Unlock()
			p.dirty()
		}
		if err != nil {
			break // EIO on Linux / EOF on macOS once the child side closes
		}
	}
	werr := cmd.Wait()
	p.mu.Lock()
	if p.gen == gen {
		p.exitedLocked(werr)
	}
	p.mu.Unlock()
	p.dirty()
}

func (p *Pane) exitedLocked(werr error) {
	p.exitedAt = time.Now()
	p.code, p.signal = 0, ""
	var ee *exec.ExitError
	if errors.As(werr, &ee) {
		p.code = ee.ExitCode()
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			p.signal = unix.SignalName(ws.Signal())
		}
	} else if werr != nil {
		p.code = -1
	}
	switch {
	case p.code == 0 && p.signal == "":
		p.state = Exited
	case p.code == 255 && strings.HasPrefix(strings.TrimSpace(p.Spec.Exec), "ssh "):
		p.state = Disconnected
	default:
		p.state = Failed
	}
	if p.ptmx != nil {
		_ = p.ptmx.Close()
		p.ptmx = nil
	}

	// Opt-in automatic restart with back-off: 1s, 2s, 4s … capped at 30s.
	pol := p.Spec.Restart
	if pol == RestartAlways || (pol == RestartOnFailure && p.state != Exited) {
		delay := time.Second << min(p.restarts, 5)
		delay = min(delay, 30*time.Second)
		p.restarts++
		gen := p.gen
		p.timer = time.AfterFunc(delay, func() {
			p.mu.Lock()
			if p.gen == gen && p.state.Dead() {
				_ = p.startLocked()
			}
			p.mu.Unlock()
			p.dirty()
		})
	}
}

// Restart relaunches the pane's launch command in the same slot. It works
// on live panes too (the old process is hung up).
func (p *Pane) Restart() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.restarts = 0
	return p.startLocked()
}

// Input sends bytes to the child as if typed. Dead panes ignore input.
func (p *Pane) Input(b []byte) {
	p.mu.Lock()
	f := p.ptmx
	alive := p.state == Running
	p.mu.Unlock()
	if alive && f != nil {
		_, _ = f.Write(b)
	}
}

// Resize changes both the PTY (the child gets SIGWINCH) and the emulator.
func (p *Pane) Resize(cols, rows int) {
	if cols < 1 || rows < 1 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if cols == p.cols && rows == p.rows {
		return
	}
	p.cols, p.rows = cols, rows
	if p.em != nil {
		p.em.Resize(cols, rows)
	}
	if p.ptmx != nil {
		_ = pty.Setsize(p.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	}
}

// Snapshot is the per-frame state the UI needs besides the cells.
type Snapshot struct {
	CursorX, CursorY int
	CursorVisible    bool
	Modes            emu.Modes
	Title            string // OSC title from the program, if any
	State            State
	Code             int
	Signal           string
	ExitedAt         time.Time
	RestartPending   bool
}

// Draw paints the pane's screen into area and returns its state.
func (p *Pane) Draw(dst uv.Screen, area uv.Rectangle) Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := Snapshot{State: p.state, Code: p.code, Signal: p.signal, ExitedAt: p.exitedAt, RestartPending: p.timer != nil}
	if p.em == nil {
		return s
	}
	p.em.Draw(dst, area)
	s.CursorX, s.CursorY, s.CursorVisible = p.em.Cursor()
	s.Modes = p.em.Modes()
	s.Title = p.em.Title()
	if s.State.Dead() {
		s.CursorVisible = false
		s.Modes = emu.Modes{}
	}
	return s
}

// Dead reports whether the pane's process has ended.
func (p *Pane) Dead() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.Dead()
}

// Close hangs up the process for good (dozer is quitting).
func (p *Pane) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
	p.stopLocked()
}
