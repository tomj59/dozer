package app

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/tomj59/dozer/internal/layout"
	"github.com/tomj59/dozer/internal/pane"
)

// view is the off-screen canvas the layout is drawn on, plus the viewport
// (DP-1): when the canvas is larger than the screen, the screen shows the
// part of it that holds the focused pane.
type view struct {
	canvas uv.ScreenBuffer
	vx, vy int
}

func (v *view) reset(w, h int) {
	v.canvas = uv.NewScreenBuffer(w, h)
}

// Chrome colors. DP-5: these become named tokens of a chrome theme
// (docs/SPEC.md §4.11); keep all chrome styling going through them.
var (
	colDim     color.Color = ansi.BasicColor(8)  // bright black: dividers
	colFocus   color.Color = ansi.BasicColor(14) // bright cyan: focused title
	colFail    color.Color = ansi.BasicColor(9)  // bright red: failed panes
	colBarText color.Color = ansi.BasicColor(15)
)

// render composes one frame and presents it (main loop).
func (a *App) render() {
	cv := a.view.canvas
	focus := int(a.focus.Load())
	slots := a.visibleSlots()

	var cursorX, cursorY int
	cursorOn := false
	dead := 0
	var focusSnap pane.Snapshot
	for i, s := range slots {
		c := s.Content()
		area := uv.Rect(c.X, c.Y, c.W, c.H)
		snap := a.panes[i].Draw(cv, area)
		if snap.State.Dead() {
			dead++
			dim(cv, area)
			a.drawBanner(cv, c, snap)
		}
		a.drawTitle(cv, s, i, snap, i == focus)
		if i == focus {
			focusSnap = snap
			cursorX, cursorY, cursorOn = c.X+snap.CursorX, c.Y+snap.CursorY, snap.CursorVisible
		}
	}
	if !a.zoomed {
		a.drawDividers(cv)
	}
	dead = 0
	for _, p := range a.panes { // hidden (zoomed-out) panes count too
		if p.Dead() {
			dead++
		}
	}

	// Viewport: keep the focused slot on screen.
	w, h := a.term.Size()
	y0, sh := a.area(w, h)
	fs := slots[focus]
	a.view.vx = follow(a.view.vx, fs.X, fs.W, w, a.res.W)
	a.view.vy = follow(a.view.vy, fs.Y, fs.H, sh, a.res.H)
	scr := a.term.Scr
	for y := 0; y < sh; y++ {
		for x := 0; x < w; x++ {
			c := cv.CellAt(a.view.vx+x, a.view.vy+y)
			if c == nil || (x == 0 && c.Width == 0 && c.Content == "") {
				scr.SetCell(x, y0+y, nil)
				continue
			}
			scr.SetCell(x, y0+y, c)
			if c.Width > 1 {
				x += c.Width - 1
			}
		}
	}

	// The bar also appears (over the bottom row) while hidden if dozer
	// needs to say something: a prefix, question or prompt.
	modal := a.mode.Load() != modeNormal || a.prefixShown()
	if a.barOn || modal {
		cx, ok := a.drawStatus(scr, a.barRow(h), w, focusSnap, dead)
		if ok { // text prompt: put the cursor in it
			cursorX, cursorY, cursorOn = cx+a.view.vx, a.barRow(h)-y0+a.view.vy, true
		}
	}

	// dozer takes the host mouse itself (button + drag reports, SGR) and
	// passes events on per pane; any-motion tracking only while the
	// focused program wants hover events.
	m := focusSnap.Modes
	any := m.MouseAny
	m.MouseX10, m.MouseNormal, m.MouseButton, m.MouseAny, m.MouseSGR = false, false, false, false, false
	if a.cfg.Mouse {
		m.MouseButton, m.MouseSGR, m.MouseAny = true, true, any
	}
	a.term.Mirror(m)

	show := cursorOn && !a.prefixShown() && a.mode.Load() != modeConfirm
	_ = a.term.Present(cursorX-a.view.vx, y0+cursorY-a.view.vy, show)
}

// follow returns a viewport offset that keeps [pos, pos+size) visible.
func follow(off, pos, size, screen, canvas int) int {
	if canvas <= screen {
		return 0
	}
	if pos < off || size > screen {
		off = pos
	} else if pos+size > off+screen {
		off = pos + size - screen
	}
	return max(0, min(off, canvas-screen))
}

func dim(cv uv.Screen, r uv.Rectangle) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if c := cv.CellAt(x, y); c != nil {
				d := *c
				d.Style.Attrs |= uv.AttrFaint
				cv.SetCell(x, y, &d)
			}
		}
	}
}

// put writes s at (x, y), clipped to maxX, and returns the next x.
func put(cv uv.Screen, x, y, maxX int, s string, st uv.Style) int {
	for _, r := range s {
		w := ansi.StringWidth(string(r))
		if w < 1 {
			w = 1
		}
		if x+w > maxX {
			break
		}
		cv.SetCell(x, y, &uv.Cell{Content: string(r), Width: w, Style: st})
		x += w
	}
	return x
}

func stateText(s pane.Snapshot) string {
	// A pane killed with C-a x says so: shells catch SIGHUP and exit with
	// their own status (zsh exits 1), which would otherwise read as a crash.
	if s.Killed && s.State.Dead() {
		how := fmt.Sprintf("exit %d", s.Code)
		if s.Signal != "" {
			how = s.Signal
		}
		return "✖ killed (" + how + ")"
	}
	switch s.State {
	case pane.Exited:
		return "exited"
	case pane.Disconnected:
		return "✖ connection lost"
	case pane.Failed:
		if s.Signal != "" {
			return "✖ " + s.Signal
		}
		return fmt.Sprintf("✖ exit %d", s.Code)
	}
	return ""
}

// drawTitle draws a slot's title divider: "─ [2] label · state ────".
func (a *App) drawTitle(cv uv.Screen, s layout.Rect, i int, snap pane.Snapshot, focused bool) {
	line := uv.Style{Fg: colDim}
	text := uv.Style{Fg: colDim}
	switch {
	case snap.State == pane.Failed || snap.State == pane.Disconnected:
		line.Fg, text.Fg = colFail, colFail
		text.Attrs |= uv.AttrBold
	case snap.State == pane.Exited:
		line.Attrs |= uv.AttrFaint
		text.Attrs |= uv.AttrFaint
	}
	if focused {
		line.Fg = colFocus
		if !snap.State.Dead() || snap.State == pane.Exited {
			text.Fg = colFocus
		}
		text.Attrs |= uv.AttrBold
	}
	end := s.X + s.W
	for x := s.X; x < end; x++ {
		cv.SetCell(x, s.Y, &uv.Cell{Content: "─", Width: 1, Style: line})
	}
	label := a.panes[i].Label()
	if snap.Title != "" && a.panes[i].Spec.Title == "" {
		label = snap.Title // the program's own title, unless the config named the pane
	}
	t := fmt.Sprintf(" [%d] %s ", i+1, label)
	if st := stateText(snap); st != "" {
		t = fmt.Sprintf(" [%d] %s · %s ", i+1, label, st)
	}
	if a.zoomed {
		t += "(zoomed) "
	}
	put(cv, s.X+1, s.Y, end-1, t, text)
}

// drawBanner puts the dead-pane banner on the pane's last content row.
func (a *App) drawBanner(cv uv.Screen, c layout.Rect, s pane.Snapshot) {
	if c.H < 1 {
		return
	}
	st := uv.Style{Fg: colBarText, Bg: colFail, Attrs: uv.AttrBold}
	if s.State == pane.Exited {
		st = uv.Style{Attrs: uv.AttrReverse}
	}
	msg := fmt.Sprintf(" %s at %s · C-%c r restart ", strings.TrimPrefix(stateText(s), "✖ "), s.ExitedAt.Format("15:04:05"), 'a'+a.cfg.Prefix-1)
	if s.State != pane.Exited {
		msg = " ✖" + msg
	}
	switch {
	case s.RestartPending && s.MaxRestarts >= 0:
		msg = fmt.Sprintf(" ↻ restarting automatically (%d/%d)… ", s.Restarts, s.MaxRestarts)
	case s.RestartPending:
		msg = fmt.Sprintf(" ↻ restarting automatically (#%d)… ", s.Restarts)
	case s.GaveUp:
		msg = fmt.Sprintf(" ✖ %s · gave up after %d automatic restarts · C-%c r to try again ",
			strings.TrimPrefix(stateText(s), "✖ "), s.Restarts, 'a'+a.cfg.Prefix-1)
	}
	y := c.Y + c.H - 1
	for x := c.X; x < c.X+c.W; x++ {
		cv.SetCell(x, y, &uv.Cell{Content: " ", Width: 1, Style: st})
	}
	put(cv, c.X, y, c.X+c.W, msg, st)
}

// drawDividers draws vertical dividers, joining them to title rows.
func (a *App) drawDividers(cv uv.Screen) {
	isTitle := func(x, y int) bool {
		for _, s := range a.res.Slots {
			if y == s.Y && x >= s.X && x < s.X+s.W {
				return true
			}
		}
		return false
	}
	st := uv.Style{Fg: colDim}
	for _, d := range a.res.Dividers {
		for y := d.Y; y < d.Y+d.H; y++ {
			up, down := y > d.Y, y < d.Y+d.H-1
			l, r := isTitle(d.X-1, y), isTitle(d.X+1, y)
			cv.SetCell(d.X, y, &uv.Cell{Content: boxChar(up, down, l, r), Width: 1, Style: st})
		}
	}
}

func boxChar(up, down, l, r bool) string {
	switch {
	case up && down && l && r:
		return "┼"
	case up && down && l:
		return "┤"
	case up && down && r:
		return "├"
	case down && l && r:
		return "┬"
	case up && l && r:
		return "┴"
	case down && r:
		return "┌"
	case down && l:
		return "┐"
	}
	return "│"
}

// drawStatus draws the bottom status bar.
// drawStatus draws the status bar on row y. For a text prompt it returns
// the cursor column and true.
func (a *App) drawStatus(scr uv.Screen, y, w int, focusSnap pane.Snapshot, dead int) (int, bool) {
	base := uv.Style{Attrs: uv.AttrReverse}
	for x := 0; x < w; x++ {
		scr.SetCell(x, y, &uv.Cell{Content: " ", Width: 1, Style: base})
	}
	pfx := fmt.Sprintf("C-%c", 'a'+a.cfg.Prefix-1)
	focus := int(a.focus.Load())
	var left string
	loud := true
	switch {
	case a.mode.Load() == modeConfirm:
		left = " " + a.confirmMsg + "  y = yes, any other key = no "
	case a.mode.Load() == modePrompt:
		left = " " + a.promptMsg + " " + string(a.promptBuf)
		x := put(scr, 0, y, w, left, withBold(base, true))
		hint := a.promptHint
		if hint == "" {
			hint = "Enter = save · Esc = cancel · empty = default"
		}
		put(scr, x+1, y, w, "  "+hint+" ", base)
		return x, true
	case a.prefixShown() && a.armedUntil.Load() != 0:
		left = " RESIZE │ H J K L again to keep resizing · = reset sizes · any other key to finish "
	case focusSnap.Copy && !a.prefixShown():
		left = fmt.Sprintf(" COPY [%d] │ %d/%d │ hjkl move · v/V select · y copy · / ? search · n N · q exit ",
			focus+1, focusSnap.Scrolled, focusSnap.History)
		if a.flash != "" && time.Since(a.flashAt) < 2*time.Second {
			left = fmt.Sprintf(" COPY [%d] │ %s ", focus+1, a.flash)
		}
	case a.prefixShown():
		left = " PREFIX │ ←↑↓→ hjkl 1-9 o focus · z zoom · [ copy · HJKL resize · = reset · r/R restart · x kill · t title · s bar · C-l redraw · q quit · Esc "
	default:
		loud = false
		name := a.cfg.Name
		if name == "" {
			name = "dozer"
		}
		left = fmt.Sprintf(" %s │ [%d] %s ", name, focus+1, a.panes[focus].Label())
		if a.zoomed {
			left += "│ ZOOM "
		}
		if a.flash != "" && time.Since(a.flashAt) < 2*time.Second {
			left += "│ " + a.flash + " "
		}
	}
	x := put(scr, 0, y, w, left, withBold(base, loud))

	// Right side: segments in priority order; the leftmost-listed survive
	// on narrow screens. Dead panes are loud (red); the rest is calm.
	type seg struct {
		text string
		st   uv.Style
	}
	var segs []seg
	calm := !loud
	if dead > 0 && calm {
		segs = append(segs, seg{fmt.Sprintf(" ✖ %d dead · %s R ", dead, pfx), uv.Style{Fg: colBarText, Bg: colFail, Attrs: uv.AttrBold}})
	}
	if hint := a.scrollHint(); hint != "" {
		segs = append(segs, seg{" " + hint + " ", withBold(base, true)})
	}
	if calm {
		segs = append(segs, seg{fmt.Sprintf(" %s q quit ", pfx), base}, seg{"│ " + a.opt.Version + " ", base})
	}
	// Keep as many as fit after the left text, dropping from the end.
	for len(segs) > 0 {
		tw := 0
		for _, sg := range segs {
			tw += ansi.StringWidth(sg.text)
		}
		if x+tw <= w {
			rx := w - tw
			for _, sg := range segs {
				rx = put(scr, rx, y, w, sg.text, sg.st)
			}
			break
		}
		segs = segs[:len(segs)-1]
	}
	return 0, false
}

func withBold(s uv.Style, on bool) uv.Style {
	if on {
		s.Attrs |= uv.AttrBold
	}
	return s
}

// scrollHint shows which directions have off-screen canvas (DP-1).
func (a *App) scrollHint() string {
	w, h := a.term.Size()
	_, sh := a.area(w, h)
	var b strings.Builder
	if a.view.vx > 0 {
		b.WriteString("◀")
	}
	if a.view.vy > 0 {
		b.WriteString("▲")
	}
	if a.view.vy+sh < a.res.H {
		b.WriteString("▼")
	}
	if a.view.vx+w < a.res.W {
		b.WriteString("▶")
	}
	if b.Len() == 0 {
		return ""
	}
	return "more " + b.String()
}
