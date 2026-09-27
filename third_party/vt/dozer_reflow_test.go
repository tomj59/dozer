package vt

import (
	"strings"
	"testing"
)

// rows returns the screen as trimmed text lines.
func rows(e *Emulator) []string {
	var out []string
	for y := 0; y < e.Height(); y++ {
		var b strings.Builder
		for x := 0; x < e.Width(); x++ {
			c := e.CellAt(x, y)
			if c == nil || c.Width == 0 && c.Content == "" {
				continue
			}
			if c.Content == "" {
				b.WriteByte(' ')
			} else {
				b.WriteString(c.Content)
			}
		}
		out = append(out, strings.TrimRight(b.String(), " "))
	}
	return out
}

func history(e *Emulator) []string {
	var out []string
	for _, l := range e.Scrollback().Lines() {
		var b strings.Builder
		for _, c := range l {
			if c.Width == 0 && c.Content == "" {
				continue
			}
			b.WriteString(c.Content)
		}
		out = append(out, strings.TrimRight(b.String(), " "))
	}
	return out
}

func TestWrapMarked(t *testing.T) {
	e := NewEmulator(10, 4)
	e.WriteString("0123456789abc\r\nnext")
	if !LineWrapped(e.scr.buf.Line(0)) {
		t.Fatal("row 0 should be marked wrapped")
	}
	if LineWrapped(e.scr.buf.Line(1)) || LineWrapped(e.scr.buf.Line(2)) {
		t.Fatal("rows 1-2 must not be marked")
	}
}

func TestReflowShrinkThenGrowRestores(t *testing.T) {
	e := NewEmulator(20, 5)
	e.WriteString("hello wide world!!\r\nline two\r\n$ ")
	e.Resize(8, 5)
	got := rows(e)
	want := []string{"hello wi", "de world", "!!", "line two", "$"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("shrunk:\n got %q\nwant %q", got, want)
	}
	if x, y := e.scr.CursorPosition(); x != 2 || y != 4 {
		t.Errorf("cursor after shrink = %d,%d, want 2,4", x, y)
	}
	e.Resize(20, 5)
	got = rows(e)
	want = []string{"hello wide world!!", "line two", "$", "", ""}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("regrown:\n got %q\nwant %q", got, want)
	}
	if x, y := e.scr.CursorPosition(); x != 2 || y != 2 {
		t.Errorf("cursor after regrow = %d,%d, want 2,2", x, y)
	}
}

func TestReflowPushesToScrollbackAndPullsBack(t *testing.T) {
	e := NewEmulator(10, 3)
	e.WriteString("a\r\nb\r\nc")
	e.Resize(10, 2) // shorter: top line goes to history, not lost
	if h := history(e); len(h) != 1 || h[0] != "a" {
		t.Fatalf("history = %q, want [a]", h)
	}
	if got := rows(e); got[0] != "b" || got[1] != "c" {
		t.Fatalf("screen = %q", got)
	}
	e.Resize(10, 3) // taller: it comes back
	if got := rows(e); strings.Join(got, "|") != "a|b|c" {
		t.Fatalf("screen = %q", got)
	}
	if e.ScrollbackLen() != 0 {
		t.Fatalf("history should be empty, has %d", e.ScrollbackLen())
	}
}

func TestReflowWideGlyphs(t *testing.T) {
	e := NewEmulator(10, 3)
	e.WriteString("日本語テキスト") // 14 columns
	e.Resize(5, 4)
	got := rows(e)
	want := []string{"日本", "語テ", "キス", "ト"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAltScreenNotReflowed(t *testing.T) {
	e := NewEmulator(10, 3)
	e.WriteString("keep me here\r\n")
	e.WriteString("\x1b[?1049h")
	e.WriteString("ALT")
	e.Resize(5, 3)
	if got := rows(e); got[0] != "ALT" {
		t.Fatalf("alt screen = %q", got)
	}
	e.WriteString("\x1b[?1049l")
	e.Resize(12, 3)
	if got := rows(e); got[0] != "keep me here" {
		t.Fatalf("main screen after alt = %q", got)
	}
}

func TestDrawStripsWrapMark(t *testing.T) {
	e := NewEmulator(4, 2)
	e.WriteString("abcdef")
	c := e.CellAt(3, 0)
	if !isWrapCell(c) {
		t.Fatal("expected marker on last cell")
	}
	if s := StripWrap(c); isWrapCell(s) || s.Content != "d" {
		t.Fatalf("StripWrap = %+v", s)
	}
}
