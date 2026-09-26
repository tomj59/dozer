package input

import (
	"bytes"
	"testing"
)

func collect(r *Router, chunks ...string) (fwd string, cmds string) {
	var f bytes.Buffer
	var c bytes.Buffer
	for _, ch := range chunks {
		for _, a := range r.Feed([]byte(ch)) {
			if a.Kind == Forward {
				f.Write(a.Data)
			} else {
				c.WriteByte(a.Key)
			}
		}
	}
	return f.String(), c.String()
}

func TestRouter(t *testing.T) {
	tests := []struct {
		name   string
		chunks []string
		fwd    string
		cmds   string
	}{
		{"plain", []string{"ls -la\r"}, "ls -la\r", ""},
		{"command", []string{"ab\x01qcd"}, "abcd", "q"},
		{"prefix split across reads", []string{"ab\x01", "zcd"}, "abcd", "z"},
		{"literal prefix", []string{"a\x01\x01b"}, "a\x01b", ""},
		{"escape sequences pass", []string{"\x1b[A\x1b[1;5C\x1bx"}, "\x1b[A\x1b[1;5C\x1bx", ""},
		{"esc cancels prefix", []string{"\x01\x1b", "ab"}, "ab", ""},
		{"arrow after prefix is swallowed", []string{"x\x01\x1b[Ay"}, "xy", ""},
		{"modified arrow after prefix", []string{"\x01\x1b[1;5Cz"}, "z", ""},
		{"ss3 arrow after prefix", []string{"\x01\x1bOAz"}, "z", ""},
		{"lone esc then key in same read", []string{"\x01\x1bq"}, "q", ""},
		{"paste keeps prefix byte", []string{"\x1b[200~x\x01q\x1b[201~"}, "\x1b[200~x\x01q\x1b[201~", ""},
		{"paste markers split", []string{"\x1b[2", "00~\x01", "\x1b[20", "1~\x01z"}, "\x1b[200~\x01\x1b[201~", "z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fwd, cmds := collect(NewRouter(0x01), tt.chunks...)
			if fwd != tt.fwd || cmds != tt.cmds {
				t.Fatalf("got fwd=%q cmds=%q, want fwd=%q cmds=%q", fwd, cmds, tt.fwd, tt.cmds)
			}
		})
	}
}
