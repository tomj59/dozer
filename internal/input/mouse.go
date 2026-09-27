package input

import "strconv"

// Mouse button codes (SGR mouse, xterm).
const (
	MouseLeft      = 0
	MouseMiddle    = 1
	MouseRight     = 2
	MouseNone      = 3 // motion with no button pressed
	MouseWheelUp   = 64
	MouseWheelDown = 65
)

// MouseEvent is one decoded SGR mouse report from the host terminal.
type MouseEvent struct {
	Button  int  // MouseLeft … MouseWheelDown
	X, Y    int  // 0-based screen cell
	Release bool // button released
	Motion  bool // pointer moved (drag, or hover with MouseNone)
	Shift   bool
	Alt     bool
	Ctrl    bool
}

// Wheel reports whether e is a scroll-wheel event.
func (e MouseEvent) Wheel() bool { return e.Button >= 64 && e.Button < 128 }

// MouseFilter removes SGR mouse reports (ESC [ < b ; x ; y M/m) from host
// input, so the rest can go through the Router untouched. A report split
// across reads is held until it completes.
type MouseFilter struct {
	held []byte
}

// Feed returns the input without mouse reports, and the reports.
func (f *MouseFilter) Feed(b []byte) ([]byte, []MouseEvent) {
	if len(f.held) > 0 {
		b = append(f.held, b...)
		f.held = nil
	}
	var rest []byte
	var events []MouseEvent
	for i := 0; i < len(b); {
		if b[i] != 0x1b || !hasPrefix(b[i:], "\x1b[<") {
			// Not a report. (A read ending in a bare "ESC" or "ESC [" is
			// passed on: keys arrive whole, so that is a key, not the
			// start of a split report.)
			rest = append(rest, b[i])
			i++
			continue
		}
		j := i + 3
		for j < len(b) && (b[j] >= '0' && b[j] <= '9' || b[j] == ';') {
			j++
		}
		if j >= len(b) { // incomplete report: wait for the rest
			f.held = append([]byte(nil), b[i:]...)
			break
		}
		if ev, ok := parseSGR(string(b[i+3:j]), b[j]); ok {
			events = append(events, ev)
			i = j + 1
			continue
		}
		rest = append(rest, b[i])
		i++
	}
	return rest, events
}

func hasPrefix(b []byte, p string) bool { return len(b) >= len(p) && string(b[:len(p)]) == p }

func parseSGR(params string, final byte) (MouseEvent, bool) {
	if final != 'M' && final != 'm' {
		return MouseEvent{}, false
	}
	var n [3]int
	k := 0
	start := 0
	for i := 0; i <= len(params); i++ {
		if i == len(params) || params[i] == ';' {
			if k >= 3 {
				return MouseEvent{}, false
			}
			v, err := strconv.Atoi(params[start:i])
			if err != nil {
				return MouseEvent{}, false
			}
			n[k] = v
			k++
			start = i + 1
		}
	}
	if k != 3 {
		return MouseEvent{}, false
	}
	b := n[0]
	return MouseEvent{
		Button:  b &^ (4 | 8 | 16 | 32),
		X:       n[1] - 1,
		Y:       n[2] - 1,
		Release: final == 'm',
		Motion:  b&32 != 0,
		Shift:   b&4 != 0,
		Alt:     b&8 != 0,
		Ctrl:    b&16 != 0,
	}, true
}

// Encode renders e for a program in a pane, at pane-relative x, y, in the
// program's requested format (SGR, or the legacy X10 byte form).
func (e MouseEvent) Encode(x, y int, sgr bool) []byte {
	b := e.Button
	if e.Shift {
		b |= 4
	}
	if e.Alt {
		b |= 8
	}
	if e.Ctrl {
		b |= 16
	}
	if e.Motion {
		b |= 32
	}
	if sgr {
		final := byte('M')
		if e.Release {
			final = 'm'
		}
		return []byte("\x1b[<" + strconv.Itoa(b) + ";" + strconv.Itoa(x+1) + ";" + strconv.Itoa(y+1) + string(final))
	}
	if e.Release {
		b = 3 | b&^3 // legacy: release has no button
		if e.Wheel() {
			return nil
		}
	}
	if x > 222 || y > 222 { // the legacy form can't express it
		return nil
	}
	return []byte{0x1b, '[', 'M', byte(32 + b), byte(33 + x), byte(33 + y)}
}
