package input

import (
	"reflect"
	"testing"
)

func TestMouseFilter(t *testing.T) {
	var f MouseFilter
	rest, ev := f.Feed([]byte("ab\x1b[<0;10;5Mc\x1b[<32;11;5M\x1b[<0;11;5m\x1b[A"))
	if string(rest) != "abc\x1b[A" {
		t.Fatalf("rest = %q", rest)
	}
	want := []MouseEvent{
		{Button: 0, X: 9, Y: 4},
		{Button: 0, X: 10, Y: 4, Motion: true},
		{Button: 0, X: 10, Y: 4, Release: true},
	}
	if !reflect.DeepEqual(ev, want) {
		t.Fatalf("events = %+v", ev)
	}
	// Split across reads.
	rest, ev = f.Feed([]byte("x\x1b[<65;3"))
	if string(rest) != "x" || len(ev) != 0 {
		t.Fatalf("first half: %q %v", rest, ev)
	}
	rest, ev = f.Feed([]byte(";4My"))
	if string(rest) != "y" || len(ev) != 1 || ev[0].Button != MouseWheelDown || !ev[0].Wheel() || ev[0].X != 2 {
		t.Fatalf("second half: %q %+v", rest, ev)
	}
	// Modifiers.
	_, ev = f.Feed([]byte("\x1b[<20;1;1M"))
	if len(ev) != 1 || !ev[0].Shift || !ev[0].Ctrl || ev[0].Button != 0 {
		t.Fatalf("mods: %+v", ev)
	}
	// A lone Esc passes through.
	if rest, _ := f.Feed([]byte("\x1b")); string(rest) != "\x1b" {
		t.Fatalf("lone esc: %q", rest)
	}
}

func TestMouseEncode(t *testing.T) {
	e := MouseEvent{Button: MouseLeft, Release: true}
	if got := string(e.Encode(4, 2, true)); got != "\x1b[<0;5;3m" {
		t.Fatalf("sgr = %q", got)
	}
	if got := e.Encode(4, 2, false); !reflect.DeepEqual(got, []byte{0x1b, '[', 'M', 32 + 3, 33 + 4, 33 + 2}) {
		t.Fatalf("x10 = %v", got)
	}
	d := MouseEvent{Button: MouseLeft, Motion: true}
	if got := string(d.Encode(0, 0, true)); got != "\x1b[<32;1;1M" {
		t.Fatalf("drag = %q", got)
	}
}
