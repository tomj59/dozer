// Package app wires the host terminal, input routing and panes together.
//
// M0 (spike): a single full-screen pane with a one-line status bar at the
// bottom. The pane sits at the screen origin so raw mouse reports from the
// host need no coordinate translation yet.
package app

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/tomj59/dozer/internal/emu"
	"github.com/tomj59/dozer/internal/host"
	"github.com/tomj59/dozer/internal/input"
	"github.com/tomj59/dozer/internal/pane"
)

// Options configure a run.
type Options struct {
	Emulator string    // emulator back end (DP-2 spike)
	Spec     pane.Spec // what the pane launches
	Prefix   byte      // prefix key byte
	Version  string
}

const frame = time.Second / 60 // render at most ~60 fps

// Run starts dozer and blocks until it exits. It returns the pane's exit code.
func Run(o Options) (code int, err error) {
	if o.Prefix == 0 {
		o.Prefix = 0x01
	}
	term, err := host.Open()
	if err != nil {
		return 1, fmt.Errorf("open terminal: %w", err)
	}
	defer term.Restore()
	defer func() { // never leave the user's terminal in raw mode
		if r := recover(); r != nil {
			term.Restore()
			panic(r)
		}
	}()

	w, h := term.Size()
	em, err := emu.New(o.Emulator, w, paneRows(h))
	if err != nil {
		return 1, err
	}

	dirty := make(chan struct{}, 1)
	mark := func() {
		select {
		case dirty <- struct{}{}:
		default:
		}
	}

	p, err := pane.Start(1, o.Spec, em, w, paneRows(h), mark)
	if err != nil {
		return 1, fmt.Errorf("start pane: %w", err)
	}
	defer p.Close()

	quit := make(chan struct{})
	redraw := make(chan struct{}, 1)
	router := input.NewRouter(o.Prefix)
	var prefixPending atomic.Bool

	// Input: raw bytes from the host keyboard.
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := term.In.Read(buf)
			if err != nil {
				close(quit)
				return
			}
			for _, a := range router.Feed(buf[:n]) {
				switch a.Kind {
				case input.Forward:
					p.Input(a.Data)
				case input.Command:
					switch a.Key {
					case 'q':
						close(quit)
						return
					case 'l': // repaint everything (e.g. after Cmd-K cleared the host screen)
						select {
						case redraw <- struct{}{}:
						default:
						}
					}
				}
			}
			prefixPending.Store(router.Pending())
			mark()
		}
	}()

	// Resize: SIGWINCH from the host.
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)

	status := statusBar{emu: o.Emulator, version: o.Version, prefix: o.Prefix}
	if status.emu == "" {
		status.emu = "charm"
	}

	render := func() {
		scr := term.Scr
		bw, bh := scr.Bounds().Dx(), scr.Bounds().Dy()
		area := uv.Rect(0, 0, bw, paneRows(bh))
		snap := p.Draw(scr, area)
		status.pending = prefixPending.Load()
		status.title = snap.Title
		status.draw(scr, bh-1, bw)
		term.Mirror(snap.Modes)
		if snap.CursorVisible && !status.pending {
			_ = scr.SetCursorPosition(area.Min.X+snap.CursorX, area.Min.Y+snap.CursorY)
			_ = scr.ShowCursor()
		} else {
			_ = scr.HideCursor()
		}
		_ = scr.Render()
		_ = scr.Flush()
	}

	var last time.Time
	for {
		select {
		case <-quit:
			return 0, nil
		case <-p.Done():
			return p.ExitCode(), nil
		case <-redraw:
			w, h := term.Size()
			_ = term.Scr.Resize(w, h) // Resize erases, forcing a full repaint
			mark()
		case <-winch:
			w, h := term.Size()
			_ = term.Scr.Resize(w, h)
			p.Resize(w, paneRows(h))
			mark()
		case <-dirty:
			if d := time.Since(last); d < frame {
				time.Sleep(frame - d) // coalesce bursts of output into one frame
			}
			render()
			last = time.Now()
		}
	}
}

func paneRows(h int) int {
	if h < 2 {
		return 1
	}
	return h - 1 // last row is the status bar
}

type statusBar struct {
	emu, version, title string
	prefix              byte
	pending             bool
}

func (s statusBar) draw(scr uv.Screen, y, width int) {
	style := uv.Style{Attrs: uv.AttrReverse}
	left := fmt.Sprintf(" dozer %s │ emu: %s │ C-%c q: quit ", s.version, s.emu, 'a'+s.prefix-1)
	if s.pending {
		left = fmt.Sprintf(" dozer %s │ PREFIX │ q: quit  l: redraw  C-%c: send C-%c ", s.version, 'a'+s.prefix-1, 'a'+s.prefix-1)
		style.Attrs |= uv.AttrBold
	}
	line := left
	if s.title != "" {
		line += "│ " + s.title + " "
	}
	x := 0
	for _, r := range line {
		if x >= width {
			break
		}
		c := uv.Cell{Content: string(r), Width: 1, Style: style}
		scr.SetCell(x, y, &c)
		x++
	}
	for ; x < width; x++ {
		c := uv.Cell{Content: " ", Width: 1, Style: style}
		scr.SetCell(x, y, &c)
	}
}
