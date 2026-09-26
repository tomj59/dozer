// Package input splits raw host keyboard bytes into pass-through data for
// the focused pane and dozer commands.
//
// dozer does not decode and re-encode keys. Bytes pass through untouched,
// and only the prefix key is intercepted. That keeps Alt/Meta, modified
// arrows and other exotic sequences exact. (The host terminal is put into
// the focused program's key modes, so the bytes already arrive encoded
// correctly for it.)
package input

import "time"

// Kind is the kind of an Action.
type Kind int

const (
	// Forward means send Data to the focused pane.
	Forward Kind = iota
	// Command means run the dozer command bound to Key.
	Command
)

// Arrow keys pressed as command keys are reported with these Key values.
const (
	KeyUp byte = 0x80 + iota
	KeyDown
	KeyRight
	KeyLeft
)

// Action is one routed piece of input.
type Action struct {
	Kind Kind
	Data []byte // Forward
	Key  byte   // Command: the byte pressed after the prefix
}

var (
	pasteStart = []byte("\x1b[200~")
	pasteEnd   = []byte("\x1b[201~")
)

// Router is a small state machine: normal → prefix-pending → normal.
// Bracketed-paste content is never scanned for the prefix, so pasted text
// containing the prefix byte (for example ^A) passes through intact.
type Router struct {
	Prefix byte // default 0x01 (Ctrl-a)

	pending bool // prefix seen, waiting for the command key
	skipSeq int  // >0: consuming an escape sequence pressed as a command key
	seqLen  int  // bytes of that sequence after ESC [ / ESC O
	inPaste bool
	match   int // bytes of pasteStart/pasteEnd matched so far

	// Repeat: after a repeatable command (e.g. resize), the prefix stays
	// armed until armedUntil so the key can be pressed again without it.
	armedUntil time.Time
	Now        func() time.Time // for tests; default time.Now
}

// Arm keeps the prefix pending until d from now, so a repeatable command
// key can be pressed again without the prefix (tmux's repeat-time).
func (r *Router) Arm(d time.Duration) {
	r.pending = true
	r.armedUntil = r.now().Add(d)
}

// Armed reports whether the prefix is pending because of Arm.
func (r *Router) Armed() bool {
	return r.pending && !r.armedUntil.IsZero() && r.now().Before(r.armedUntil)
}

func (r *Router) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

// NewRouter returns a router for the given prefix byte.
func NewRouter(prefix byte) *Router { return &Router{Prefix: prefix} }

// Pending reports whether the prefix was pressed and a command key is awaited.
func (r *Router) Pending() bool {
	r.expire()
	return r.pending
}

// expire drops an Arm'ed prefix whose repeat window has passed.
func (r *Router) expire() {
	if !r.armedUntil.IsZero() && !r.now().Before(r.armedUntil) {
		r.pending = false
		r.armedUntil = time.Time{}
	}
}

// Feed routes a chunk of raw input.
func (r *Router) Feed(b []byte) []Action {
	r.expire()
	var out []Action
	start := 0 // start of the current forward run
	flush := func(end int) {
		if end > start {
			out = append(out, Action{Kind: Forward, Data: append([]byte(nil), b[start:end]...)})
		}
	}
	for i := 0; i < len(b); i++ {
		c := b[i]
		r.trackPaste(c)
		if r.inPaste {
			continue
		}
		if r.skipSeq > 0 {
			// An Esc/arrow/function key was pressed as the command key.
			// Plain arrows become KeyUp…KeyLeft commands; any other
			// sequence (or a lone Esc) just cancels the prefix. Keys
			// arrive whole in one read, so this never spans chunks.
			if r.skipSeq == 1 {
				if c == '[' || c == 'O' {
					r.skipSeq, r.seqLen = 2, 0
					start = i + 1
					continue
				}
				r.skipSeq = 0 // a lone Esc: handle c normally below
			} else {
				r.seqLen++
				if c >= 0x40 && c <= 0x7e { // final byte ends the sequence
					r.skipSeq = 0
					if r.seqLen == 1 && c >= 'A' && c <= 'D' {
						out = append(out, Action{Kind: Command, Key: KeyUp + (c - 'A')})
					}
				}
				start = i + 1
				continue
			}
		}
		if r.pending {
			if c == r.Prefix && !r.armedUntil.IsZero() {
				// The prefix pressed during a repeat window starts a fresh
				// command rather than sending a literal prefix.
				r.armedUntil = time.Time{}
				start = i + 1
				continue
			}
			r.pending = false
			r.armedUntil = time.Time{}
			if c == 0x1b { // Esc alone, or the start of an arrow/function key: cancel
				r.skipSeq = 1
				start = i + 1
				continue
			}
			if c == r.Prefix { // prefix twice: send one literal prefix
				start = i
				continue
			}
			out = append(out, Action{Kind: Command, Key: c})
			start = i + 1
			continue
		}
		if c == r.Prefix {
			flush(i)
			r.pending = true
			start = i + 1
		}
	}
	r.skipSeq = 0
	if !r.pending {
		flush(len(b))
	}
	return out
}

// trackPaste follows ESC[200~ … ESC[201~ across chunk boundaries.
func (r *Router) trackPaste(c byte) {
	want := pasteStart
	if r.inPaste {
		want = pasteEnd
	}
	if c == want[r.match] {
		r.match++
		if r.match == len(want) {
			r.inPaste = !r.inPaste
			r.match = 0
		}
		return
	}
	r.match = 0
	if c == want[0] {
		r.match = 1
	}
}
