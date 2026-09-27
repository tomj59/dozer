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

// Automatic-restart limits. A run that stays up for HealthyRun resets the
// count, so a service that crashes once a day never runs out of restarts.
const (
	DefaultMaxRestarts = 5
	HealthyRun         = time.Minute
)

// Restart policies for automatic restarts.
const (
	RestartNever     = "never"
	RestartOnFailure = "on-failure"
	RestartAlways    = "always"
)

// Spec describes what a pane launches. See docs/SPEC.md §4.3.
type Spec struct {
	Title string   // label; default derived from the command
	Shell string   // interactive shell; default $SHELL, then /bin/sh
	Run   string   // run: command in an interactive shell, then a prompt (DP-4)
	Exec  string   // exec: the pane process is `$SHELL -lc Exec`
	Dir   string   // working directory; "" = inherit
	Env   []string // extra KEY=VALUE entries

	// As written in the config (unexpanded: "logs", "~/work", "$HOME/x"),
	// so --save/--package reproduce the author's intent, not this machine.
	RawDir  string
	RawEnv  []string
	Restart string // never | on-failure | always (automatic restarts)
	// MaxRestarts caps consecutive automatic restarts before dozer gives up
	// and leaves the pane dead: 0 = DefaultMaxRestarts, -1 = unlimited.
	MaxRestarts int
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

// Version is reported to programs in panes as TERM_PROGRAM_VERSION.
var Version = "dev"

// hostTerminalVars identify the terminal *dozer* runs in. Inside a pane,
// dozer is the terminal, so they must not leak through. For example,
// Terminal.app's TERM_SESSION_ID makes every pane's zsh restore and save
// the same window session ("Restored session: …"), and TMUX would make
// programs think they are inside tmux.
var hostTerminalVars = map[string]bool{
	"TERM_PROGRAM": true, "TERM_PROGRAM_VERSION": true, "TERM_SESSION_ID": true,
	"LC_TERMINAL": true, "LC_TERMINAL_VERSION": true, "ITERM_SESSION_ID": true, "ITERM_PROFILE": true,
	"TMUX": true, "TMUX_PANE": true, "STY": true, "WINDOW": true,
	"KITTY_WINDOW_ID": true, "KITTY_PID": true, "KITTY_LISTEN_ON": true, "KITTY_PUBLIC_KEY": true,
	"WEZTERM_PANE": true, "WEZTERM_UNIX_SOCKET": true, "WEZTERM_EXECUTABLE": true,
	"ALACRITTY_WINDOW_ID": true, "ALACRITTY_SOCKET": true, "ALACRITTY_LOG": true,
	"VTE_VERSION": true, "WT_SESSION": true, "WT_PROFILE_ID": true,
	"GHOSTTY_RESOURCES_DIR": true, "GHOSTTY_BIN_DIR": true, "GHOSTTY_SHELL_INTEGRATION_NO_SUDO": true,
	"TERMINAL_EMULATOR": true, "KONSOLE_VERSION": true, "KONSOLE_DBUS_SESSION": true,
	"TERM": true, "COLORTERM": true, "DOZER": true, "DOZER_PANE": true,
}

// ChildEnv builds a pane's environment from dozer's own: host-terminal
// variables removed, dozer's terminal identity added.
func ChildEnv(env []string, id int) []string {
	out := make([]string, 0, len(env)+7)
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		if !hostTerminalVars[k] {
			out = append(out, kv)
		}
	}
	return append(out,
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"TERM_PROGRAM=dozer",
		"TERM_PROGRAM_VERSION="+Version,
		"DOZER=1",
		"DOZER_PANE="+strconv.Itoa(id),
	)
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

	mu        sync.Mutex // guards everything below
	em        emu.Emulator
	ptmx      *os.File
	cmd       *exec.Cmd
	cols      int
	rows      int
	gen       int // incremented per start; stale exits are ignored
	state     State
	code      int    // exit status (Exited/Failed)
	signal    string // e.g. "SIGKILL" if killed by a signal
	exitedAt  time.Time
	restarts  int         // consecutive automatic restarts (for back-off)
	startedAt time.Time   // when the current process started
	gaveUp    bool        // hit MaxRestarts; waits for a manual restart
	noRestart bool        // killed on purpose: skip automatic restart once
	killed    bool        // the current death was C-a x (shown as "killed")
	timer     *time.Timer // pending automatic restart

	scrollback int       // history lines to keep (-1 = emulator default)

	// Input to the program goes through a queue and a writer goroutine, so
	// a program that stops reading can't block the caller (the UI loop).
	in     chan []byte
	inDone chan struct{} // closed when the current process is stopped
	cm         *CopyMode // scrolled-back view / copy mode; nil when live
}

// New creates a pane; call Start to launch its process. dirty is called
// (from background goroutines) whenever the pane's display changes.
func New(id int, spec Spec, emuName string, cols, rows int, dirty func()) *Pane {
	return &Pane{ID: id, Spec: spec, emuName: emuName, cols: max(cols, 1), rows: max(rows, 1), dirty: dirty, scrollback: -1}
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
	cmd.Env = append(ChildEnv(os.Environ(), p.ID), p.Spec.Env...)
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
	p.in, p.inDone = make(chan []byte, 256), make(chan struct{})
	go writeLoop(ptmx, p.in, p.inDone)
	p.cm = nil
	if h, ok := em.(emu.History); ok && p.scrollback >= 0 {
		h.SetScrollbackSize(p.scrollback)
	}
	p.state, p.code, p.signal, p.gaveUp, p.killed = Running, 0, "", false, false
	p.startedAt = time.Now()
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
	if p.inDone != nil {
		close(p.inDone)
		p.inDone, p.in = nil, nil
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
	p.killed = p.noRestart
	if p.noRestart {
		pol, p.noRestart = RestartNever, false
	}
	if time.Since(p.startedAt) >= HealthyRun {
		p.restarts = 0 // it ran fine for a while: start counting afresh
	}
	if pol == RestartAlways || (pol == RestartOnFailure && p.state != Exited) {
		if limit := p.maxRestarts(); limit >= 0 && p.restarts >= limit {
			p.gaveUp = true // stop hammering; the user decides (C-a r)
			return
		}
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

func (p *Pane) maxRestarts() int {
	switch {
	case p.Spec.MaxRestarts < 0:
		return -1
	case p.Spec.MaxRestarts == 0:
		return DefaultMaxRestarts
	}
	return p.Spec.MaxRestarts
}

// Restart relaunches the pane's launch command in the same slot. It works
// on live panes too (the old process is hung up).
func (p *Pane) Restart() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.restarts, p.noRestart = 0, false
	return p.startLocked()
}

// Kill hangs up the pane's process group and, if it is still running
// after grace, kills it. The pane stays, as a dead pane.
func (p *Pane) Kill(grace time.Duration) {
	p.mu.Lock()
	if p.state != Running || p.cmd == nil || p.cmd.Process == nil {
		p.mu.Unlock()
		return
	}
	pid, gen := p.cmd.Process.Pid, p.gen
	if p.timer != nil { // a manual kill cancels automatic restarts
		p.timer.Stop()
		p.timer = nil
	}
	p.noRestart = true
	p.mu.Unlock()
	_ = syscall.Kill(-pid, syscall.SIGHUP)
	time.AfterFunc(grace, func() {
		p.mu.Lock()
		alive := p.gen == gen && p.state == Running
		p.mu.Unlock()
		if alive {
			_ = syscall.Kill(-pid, syscall.SIGKILL)
		}
	})
}

// SetTitle renames the pane (runtime control #2 in docs/SPEC.md §4.8).
func (p *Pane) SetTitle(t string) {
	p.mu.Lock()
	p.Spec.Title = t
	p.mu.Unlock()
}

// Label returns the pane's current label.
func (p *Pane) Label() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Spec.Label()
}

// Input sends bytes to the child as if typed. Dead panes ignore input.
func (p *Pane) Input(b []byte) { p.send(b, true) }

// TryInput is Input for generated input (mouse wheel, forwarded mouse
// events): if the program has stopped reading and its queue is full, the
// input is dropped instead of waiting.
func (p *Pane) TryInput(b []byte) { p.send(b, false) }

func (p *Pane) send(b []byte, wait bool) {
	if len(b) == 0 {
		return
	}
	p.mu.Lock()
	in, done := p.in, p.inDone
	alive := p.state == Running
	p.mu.Unlock()
	if !alive || in == nil {
		return
	}
	b = append([]byte(nil), b...)
	if wait {
		select {
		case in <- b:
		case <-done:
		}
		return
	}
	select {
	case in <- b:
	case <-done:
	default: // queue full: drop
	}
}

// writeLoop copies queued input to the program's terminal.
func writeLoop(w io.Writer, in <-chan []byte, done <-chan struct{}) {
	for {
		select {
		case b := <-in:
			if _, err := w.Write(b); err != nil {
				return
			}
		case <-done:
			return
		}
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
	scrolled := 0
	if p.cm != nil {
		scrolled = p.cm.Scrolled(p.sourceLocked())
	}
	p.cols, p.rows = cols, rows
	if p.em != nil {
		p.em.Resize(cols, rows)
	}
	if p.cm != nil {
		// Reflow renumbers history; keep the same distance from the bottom.
		src := p.sourceLocked()
		row := p.cm.Y - p.cm.Top
		p.cm.Top = liveTop(src) - scrolled
		p.cm.Y = p.cm.Top + row
		p.cm.Sel = SelNone
		p.cm.clamp(src)
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
	Restarts         int // automatic restarts so far (in the current streak)
	MaxRestarts      int // the limit (-1 = unlimited)
	GaveUp           bool
	Killed           bool // ended by C-a x, however the program reported it

	// Copy mode / scrolled-back view (C-a [, mouse wheel).
	Copy     bool
	Scrolled int    // lines above the live screen
	History  int    // lines of scrollback available
	Search   string // active search term
}

// Draw paints the pane's screen into area and returns its state.
func (p *Pane) Draw(dst uv.Screen, area uv.Rectangle) Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := Snapshot{State: p.state, Code: p.code, Signal: p.signal, ExitedAt: p.exitedAt, RestartPending: p.timer != nil,
		Restarts: p.restarts, MaxRestarts: p.maxRestarts(), GaveUp: p.gaveUp, Killed: p.killed}
	if p.em == nil {
		return s
	}
	s.CursorX, s.CursorY, s.CursorVisible = p.em.Cursor()
	if p.cm != nil {
		src := p.sourceLocked()
		p.cm.clamp(src)
		p.drawCopyLocked(dst, area, src)
		s.Copy, s.Scrolled, s.History, s.Search = true, p.cm.Scrolled(src), src.Last()-src.First()+1-src.Rows(), p.cm.SearchTerm()
		s.CursorX, s.CursorY, s.CursorVisible = p.cm.X, p.cm.Y-p.cm.Top, true
	} else {
		p.em.Draw(dst, area)
	}
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
