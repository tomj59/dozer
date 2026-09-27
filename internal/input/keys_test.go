package input

import (
	"reflect"
	"testing"
)

func TestKeys(t *testing.T) {
	got := Keys([]byte("aé\x1b[A\x1bOB\x1b[5~\x1b[6~\x1b[H\x1b[4~\r\x7f\x15\x1b"))
	want := []Key{{Rune: 'a'}, {Rune: 'é'}, {Name: "up"}, {Name: "down"}, {Name: "pgup"}, {Name: "pgdn"},
		{Name: "home"}, {Name: "end"}, {Name: "enter"}, {Name: "backspace"}, {Rune: 0x15}, {Name: "esc"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
	if k := Keys([]byte("\x1b[1;5A")); len(k) != 1 || k[0].Name != "up" {
		t.Fatalf("modified arrow: %+v", k)
	}
	if k := Keys([]byte("\x1b[200~")); len(k) != 0 {
		t.Fatalf("unknown sequence should be dropped: %+v", k)
	}
}
