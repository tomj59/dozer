// Package app wires the host terminal, input routing, layout and panes
// together, and runs the event loop.
package app

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/tomj59/dozer/internal/config"
	"github.com/tomj59/dozer/internal/host"
	"github.com/tomj59/dozer/internal/input"
	"github.com/tomj59/dozer/internal/layout"
	"github.com/tomj59/dozer/internal/pane"
)

// Options configure a run beyond the launch config.
type Options struct {
	Emulator string   // emulator back end (DP-2)
	Version  string   //
	Input    *os.File // keyboard; nil = os.Stdin
}

const frame = time.Second / 60 // render at most ~60 fps

// App is a running dozer session.
type App struct {
	cfg   *config.Config
	opt   Options
	term  *host.Terminal
	panes []*pane.Pane

	// Shared with the input goroutine.
	focus      atomic.Int32
	mode       atomic.Int32 // modeNormal, modeConfirm, modePrompt
	pending    atomic.Bool  // prefix pressed
	armedUntil atomic.Int64 // unix nanos; prefix re-armed for repeats until then

	// Owned by the main loop.
	launch  *layout.Node // the launch layout, for C-a =
	zoomed  bool
	barOn   bool          // status bar visible (C-a s)
	res     layout.Result // what's on screen (zoomed or full)
	full    layout.Result // the full layout, even when zoomed
	view    view
	drag    drag // mouse drag in progress
	flash   string
	flashAt time.Time

	confirmMsg string       // modeConfirm
	confirmYes func()       //
	promptMsg  string       // modePrompt
	promptBuf  []rune       //
	promptDone func(string) //
	promptHint string       // help text after the prompt (default: rename hints)

	cmds  chan func()
	dirty chan struct{}
	quit  chan struct{}
}

// Run starts dozer and blocks until the user quits.
func Run(cfg *config.Config, o Options) (err error) {
	if o.Input == nil {
		o.Input = os.Stdin
	}
	if o.Emulator == "" {
		o.Emulator = "charm"
	}
	a := &App{cfg: cfg, opt: o, cmds: make(chan func(), 16), dirty: make(chan struct{}, 1), quit: make(chan struct{})}
	a.launch = cfg.Layout.Clone()
	a.barOn = cfg.StatusBar != "off"

	a.term, err = host.Open(o.Input)
	if err != nil {
		return fmt.Errorf("open terminal: %w", err)
	}
	defer a.term.Restore()
	defer func() { // never leave the user's terminal in raw mode
		if r := recover(); r != nil {
			a.term.Restore()
			panic(r)
		}
	}()

	// Build the permanent pane slots, then size and start them.
	for i, spec := range cfg.Panes {
		// Every pane can see which workspace it belongs to (e.g. a guide
		// pane can print $DOZER_DESCRIPTION).
		spec.Env = append(append([]string(nil), spec.Env...),
			"DOZER_NAME="+cfg.Name, "DOZER_DESCRIPTION="+cfg.Description)
		p := pane.New(i+1, spec, o.Emulator, 80, 24, a.mark)
		p.SetScrollback(cfg.Scrollback)
		a.panes = append(a.panes, p)
	}
	defer func() {
		for _, p := range a.panes {
			p.Close()
		}
	}()
	a.relayout()
	for _, p := range a.panes {
		_ = p.Start() // a failed start shows up as a dead pane
	}

	go a.readInput()

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)

	a.mark()
	var last time.Time
	for {
		select {
		case <-a.quit:
			return nil
		case f := <-a.cmds:
			f()
			a.mark()
		case <-winch:
			a.relayout()
			a.mark()
		case <-a.dirty:
			if cfg.QuitWhenAllExited && a.allDead() {
				return nil
			}
			if d := time.Since(last); d < frame {
				time.Sleep(frame - d) // coalesce bursts of output into one frame
			}
			a.render()
			last = time.Now()
		}
	}
}

// mark schedules a repaint. Safe from any goroutine.
func (a *App) mark() {
	select {
	case a.dirty <- struct{}{}:
	default:
	}
}

// post runs f on the main loop.
func (a *App) post(f func()) {
	select {
	case a.cmds <- f:
	case <-a.quit:
	}
}

func (a *App) focused() *pane.Pane { return a.panes[a.focus.Load()] }

func (a *App) allDead() bool {
	for _, p := range a.panes {
		if !p.Dead() {
			return false
		}
	}
	return true
}

func (a *App) running() int {
	n := 0
	for _, p := range a.panes {
		if !p.Dead() {
			n++
		}
	}
	return n
}

// Input modes. Normal input goes through the prefix router; the modal
// states (a yes/no question, a text prompt) take raw keys on the main loop.
const (
	modeNormal int32 = iota
	modeConfirm
	modePrompt
)

// repeatable commands keep the prefix armed briefly (tmux's repeat-time).
const repeatWindow = 700 * time.Millisecond

func repeatable(key byte) bool { return key == 'H' || key == 'J' || key == 'K' || key == 'L' }

// readInput routes raw keyboard bytes: to the focused pane, or to commands.
func (a *App) readInput() {
	router := input.NewRouter(a.cfg.Prefix)
	var mice input.MouseFilter
	buf := make([]byte, 4096)
	for {
		n, err := a.term.In.Read(buf)
		if err != nil {
			a.post(a.doQuit)
			return
		}
		if a.mode.Load() != modeNormal {
			data := append([]byte(nil), buf[:n]...)
			a.call(func() { a.modalInput(data) })
			continue
		}
		data, events := buf[:n], []input.MouseEvent(nil)
		if a.cfg.Mouse {
			data, events = mice.Feed(data)
		}
		for _, ev := range events {
			ev := ev
			a.call(func() { a.mouse(ev) })
		}
		for _, act := range router.Feed(data) {
			switch act.Kind {
			case input.Forward:
				if a.mode.Load() != modeNormal {
					break
				}
				if p := a.focused(); p.InCopy() {
					data := act.Data
					a.call(func() { a.copyInput(data) })
				} else {
					p.Input(act.Data)
				}
			case input.Command:
				key := act.Key
				if repeatable(key) {
					router.Arm(repeatWindow)
					a.armedUntil.Store(time.Now().Add(repeatWindow).UnixNano())
					time.AfterFunc(repeatWindow+10*time.Millisecond, a.mark)
				} else {
					a.armedUntil.Store(0)
				}
				// Wait for the command, so keys typed right after a command
				// that opens a prompt or question go to that prompt.
				a.call(func() { a.command(key) })
			}
		}
		a.pending.Store(router.Pending())
		a.mark()
	}
}

// prefixShown reports whether the status bar should show PREFIX.
func (a *App) prefixShown() bool {
	if !a.pending.Load() {
		return false
	}
	until := a.armedUntil.Load()
	return until == 0 || time.Now().UnixNano() < until
}

// call runs f on the main loop and waits for it.
func (a *App) call(f func()) {
	done := make(chan struct{})
	a.post(func() { f(); close(done) })
	select {
	case <-done:
	case <-a.quit:
	}
}

// ask shows a yes/no question in the status bar; yes runs on 'y'.
func (a *App) ask(msg string, yes func()) {
	a.confirmMsg, a.confirmYes = msg, yes
	a.mode.Store(modeConfirm)
}

// prompt asks for a line of text in the status bar.
func (a *App) prompt(msg, initial string, done func(string)) {
	a.promptMsg, a.promptBuf, a.promptDone = msg, []rune(initial), done
	a.mode.Store(modePrompt)
}

// modalInput handles keys while a question or prompt is showing.
func (a *App) modalInput(b []byte) {
	switch a.mode.Load() {
	case modeConfirm:
		a.mode.Store(modeNormal)
		if len(b) > 0 && (b[0] == 'y' || b[0] == 'Y') && a.confirmYes != nil {
			a.confirmYes()
		}
	case modePrompt:
		if len(b) > 1 && b[0] == 0x1b {
			return // arrow/function keys: ignore
		}
		for len(b) > 0 {
			r, size := utf8.DecodeRune(b)
			b = b[size:]
			switch {
			case r == 0x1b || r == 0x03: // Esc / Ctrl-c: cancel
				a.mode.Store(modeNormal)
				a.promptHint = ""
				return
			case r == '\r' || r == '\n':
				a.mode.Store(modeNormal)
				a.promptHint = ""
				a.promptDone(strings.TrimSpace(string(a.promptBuf)))
				return
			case r == 0x7f || r == 0x08:
				if len(a.promptBuf) > 0 {
					a.promptBuf = a.promptBuf[:len(a.promptBuf)-1]
				}
			case r == 0x15: // Ctrl-u: clear
				a.promptBuf = a.promptBuf[:0]
			case r >= 0x20 && r != utf8.RuneError:
				a.promptBuf = append(a.promptBuf, r)
			}
		}
	}
}

func (a *App) doQuit() {
	select {
	case <-a.quit:
	default:
		close(a.quit)
	}
}

// Command keys (after the prefix). See docs/SPEC.md §4.4.
const ctrlL = 0x0c

// command runs a prefix command (main loop).
func (a *App) command(key byte) {
	switch key {
	case 'q':
		if n := a.running(); n > 0 {
			a.ask(fmt.Sprintf("Quit dozer? %d pane(s) still running will be closed.", n), a.doQuit)
		} else {
			a.doQuit()
		}
	case 'x':
		p := a.focused()
		if p.Dead() {
			a.say(fmt.Sprintf("[%d] is not running", p.ID))
			break
		}
		a.ask(fmt.Sprintf("Kill [%d] %s?", p.ID, p.Label()), func() {
			p.Kill(2 * time.Second)
			a.say(fmt.Sprintf("killed [%d]", p.ID))
		})
	case 't':
		p := a.focused()
		a.prompt(fmt.Sprintf("Title for [%d]:", p.ID), p.Label(), func(t string) {
			p.SetTitle(t) // "" restores the default label
		})
	case 's':
		a.barOn = !a.barOn
		a.relayout()
	case 'H', 'J', 'K', 'L':
		if a.zoomed {
			a.say("unzoom (C-a z) to resize")
			break
		}
		// Move the focused pane's border: 2 columns or 1 row per press.
		step := map[byte][2]int{'H': {-2, 0}, 'L': {2, 0}, 'K': {0, -1}, 'J': {0, 1}}[key]
		dx, dy := step[0], step[1]
		if layout.MoveBorder(a.cfg.Layout, int(a.focus.Load()), dx, dy, a.full, a.cfg.MinPane) {
			a.relayout()
		}
	case '=':
		a.cfg.Layout = a.launch.Clone()
		a.relayout()
		a.say("sizes reset to the launch layout")
	case ctrlL:
		a.term.Redraw()
	case '[':
		a.enterCopy()
	case 'r':
		p := a.focused()
		_ = p.Restart()
		a.say(fmt.Sprintf("restarted [%d]", p.ID))
	case 'R':
		n := 0
		for _, p := range a.panes {
			if p.Dead() {
				_ = p.Restart()
				n++
			}
		}
		a.say(fmt.Sprintf("restarted %d dead pane(s)", n))
	case 'z':
		a.zoomed = !a.zoomed
		a.relayout()
	case 'o':
		a.setFocus((int(a.focus.Load()) + 1) % len(a.panes))
	case input.KeyUp, 'k':
		a.moveFocus(0, -1)
	case input.KeyDown, 'j':
		a.moveFocus(0, 1)
	case input.KeyLeft, 'h':
		a.moveFocus(-1, 0)
	case input.KeyRight, 'l':
		a.moveFocus(1, 0)
	default:
		if key >= '1' && key <= '9' && int(key-'1') < len(a.panes) {
			a.setFocus(int(key - '1'))
		}
	}
}

func (a *App) say(msg string) {
	a.flash, a.flashAt = msg, time.Now()
	time.AfterFunc(2*time.Second, a.mark)
}

func (a *App) setFocus(i int) {
	if i == int(a.focus.Load()) {
		return
	}
	a.focus.Store(int32(i))
	if a.zoomed { // moving focus leaves zoom, like tmux
		a.zoomed = false
		a.relayout()
	}
}

// moveFocus moves to the nearest pane in direction (dx, dy).
func (a *App) moveFocus(dx, dy int) {
	if i := neighbor(a.full.Slots, int(a.focus.Load()), dx, dy); i >= 0 {
		a.setFocus(i)
	}
}

// relayout solves the layout for the current screen and resizes panes.
func (a *App) relayout() {
	w, h := a.term.Size()
	a.term.Resize(w, h)
	_, areaH := a.area(w, h)
	a.full = layout.Solve(a.cfg.Layout, w, areaH, a.cfg.MinPane)
	a.res = a.full
	if a.zoomed {
		a.res = layout.Solve(&layout.Node{Pane: 0}, w, areaH, a.cfg.MinPane)
	}
	a.view.reset(a.res.W, a.res.H)
	for i, s := range a.visibleSlots() {
		c := s.Content()
		a.panes[i].Resize(c.W, c.H)
	}
}

// area returns the first row and the height of the pane area (the screen
// minus the status bar, which sits on the top or bottom row).
func (a *App) area(w, h int) (y0, height int) {
	if !a.barOn {
		return 0, max(h, 1)
	}
	if a.cfg.StatusBar == "top" {
		return 1, max(h-1, 1)
	}
	return 0, max(h-1, 1)
}

// barRow is the status bar's row.
func (a *App) barRow(h int) int {
	if a.cfg.StatusBar == "top" {
		return 0
	}
	return h - 1
}

// visibleSlots maps pane index → slot for the panes on screen.
func (a *App) visibleSlots() map[int]layout.Rect {
	out := map[int]layout.Rect{}
	if a.zoomed {
		out[int(a.focus.Load())] = a.res.Slots[0]
		return out
	}
	for i, s := range a.res.Slots {
		out[i] = s
	}
	return out
}

// neighbor finds the pane nearest to slot cur in direction (dx, dy): it
// must lie beyond cur's edge in that direction and overlap it on the other
// axis if possible; ties go to the closest centre.
func neighbor(slots []layout.Rect, cur, dx, dy int) int {
	c := slots[cur]
	best, bestScore := -1, 1<<30
	for i, s := range slots {
		if i == cur {
			continue
		}
		var gap, overlap int
		switch {
		case dx > 0:
			gap = s.X - (c.X + c.W)
			overlap = min(c.Y+c.H, s.Y+s.H) - max(c.Y, s.Y)
		case dx < 0:
			gap = c.X - (s.X + s.W)
			overlap = min(c.Y+c.H, s.Y+s.H) - max(c.Y, s.Y)
		case dy > 0:
			gap = s.Y - (c.Y + c.H)
			overlap = min(c.X+c.W, s.X+s.W) - max(c.X, s.X)
		case dy < 0:
			gap = c.Y - (s.Y + s.H)
			overlap = min(c.X+c.W, s.X+s.W) - max(c.X, s.X)
		}
		if gap < 0 {
			continue // not in that direction
		}
		// Prefer overlapping panes, then the nearest edge, then centres.
		cx, cy := c.X+c.W/2, c.Y+c.H/2
		sx, sy := s.X+s.W/2, s.Y+s.H/2
		score := gap*1000 + abs(cx-sx) + abs(cy-sy)
		if overlap <= 0 {
			score += 1 << 20
		}
		if score < bestScore {
			best, bestScore = i, score
		}
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
