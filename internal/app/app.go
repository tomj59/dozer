// Package app wires the host terminal, input routing, layout and panes
// together, and runs the event loop.
package app

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

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
	confirming atomic.Bool // "quit? y/n" is showing
	pending    atomic.Bool // prefix pressed

	// Owned by the main loop.
	zoomed  bool
	res     layout.Result
	view    view
	flash   string
	flashAt time.Time

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
		a.panes = append(a.panes, pane.New(i+1, spec, o.Emulator, 80, 24, a.mark))
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

// readInput routes raw keyboard bytes: to the focused pane, or to commands.
func (a *App) readInput() {
	router := input.NewRouter(a.cfg.Prefix)
	buf := make([]byte, 4096)
	for {
		n, err := a.term.In.Read(buf)
		if err != nil {
			a.post(a.doQuit)
			return
		}
		if a.confirming.Load() {
			yes := buf[0] == 'y' || buf[0] == 'Y'
			a.post(func() {
				a.confirming.Store(false)
				if yes {
					a.doQuit()
				}
			})
			continue
		}
		for _, act := range router.Feed(buf[:n]) {
			switch act.Kind {
			case input.Forward:
				a.focused().Input(act.Data)
			case input.Command:
				key := act.Key
				a.post(func() { a.command(key) })
			}
		}
		a.pending.Store(router.Pending())
		a.mark()
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
		if a.running() > 0 {
			a.confirming.Store(true)
		} else {
			a.doQuit()
		}
	case ctrlL:
		a.term.Redraw()
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
	w, h := a.term.Size()
	full := layout.Solve(a.cfg.Layout, w, max(h-1, 1), a.cfg.MinPane)
	if i := neighbor(full.Slots, int(a.focus.Load()), dx, dy); i >= 0 {
		a.setFocus(i)
	}
}

// relayout solves the layout for the current screen and resizes panes.
func (a *App) relayout() {
	w, h := a.term.Size()
	a.term.Resize(w, h)
	root := a.cfg.Layout
	if a.zoomed {
		root = &layout.Node{Pane: 0}
	}
	a.res = layout.Solve(root, w, max(h-1, 1), a.cfg.MinPane) // last row: status bar
	a.view.reset(a.res.W, a.res.H)
	for i, s := range a.visibleSlots() {
		c := s.Content()
		a.panes[i].Resize(c.W, c.H)
	}
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
