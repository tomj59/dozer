package pane

import (
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
)

func start(t *testing.T, spec Spec, cols, rows int) *Pane {
	t.Helper()
	if spec.Shell == "" {
		spec.Shell = "/bin/sh"
	}
	p := New(1, spec, "charm", cols, rows, func() {})
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func draw(p *Pane, cols, rows int) (string, Snapshot) {
	buf := uv.NewScreenBuffer(cols, rows)
	s := p.Draw(buf, uv.Rect(0, 0, cols, rows))
	return buf.String(), s
}

func TestOutputSizeAndExitState(t *testing.T) {
	p := start(t, Spec{Exec: `printf 'hello \033[31mred\033[0m'; echo; printf 'size:'; stty size; exit 7`}, 40, 5)
	waitFor(t, "exit", p.Dead)
	got, s := draw(p, 40, 5)
	if !strings.Contains(got, "hello red") || !strings.Contains(got, "size:5 40") {
		t.Errorf("screen:\n%s", got)
	}
	if s.State != Failed || s.Code != 7 {
		t.Errorf("state=%v code=%d, want Failed/7", s.State, s.Code)
	}
	if !strings.Contains(got, "hello red") {
		t.Error("dead pane must keep its last screen")
	}
}

func TestExitStates(t *testing.T) {
	cases := []struct {
		exec   string
		state  State
		signal string
	}{
		{"exit 0", Exited, ""},
		{"exit 3", Failed, ""},
		{"kill -KILL $$", Failed, "SIGKILL"},
		{"ssh() { return 255; }; ssh host", Failed, ""}, // not literally `ssh …`
	}
	for _, c := range cases {
		p := start(t, Spec{Exec: c.exec}, 20, 3)
		waitFor(t, c.exec, p.Dead)
		_, s := draw(p, 20, 3)
		if s.State != c.state || s.Signal != c.signal {
			t.Errorf("%q: state=%v signal=%q, want %v %q", c.exec, s.State, s.Signal, c.state, c.signal)
		}
	}
}

func TestRestart(t *testing.T) {
	p := start(t, Spec{Exec: `echo run-$$; exit 1`}, 30, 3)
	waitFor(t, "first exit", p.Dead)
	first, _ := draw(p, 30, 3)
	if err := p.Restart(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "second exit", func() bool {
		s, _ := draw(p, 30, 3)
		return p.Dead() && s != first && strings.Contains(s, "run-")
	})
}

func TestAutoRestartOnFailure(t *testing.T) {
	p := start(t, Spec{Exec: `echo x >> "$F"; exit 1`, Restart: RestartOnFailure,
		Env: []string{"F=" + t.TempDir() + "/count"}}, 20, 3)
	// First run, then one automatic restart after the 1s back-off.
	waitFor(t, "auto restart", func() bool {
		_, s := draw(p, 20, 3)
		return p.restartsSoFar() >= 2 && s.State.Dead()
	})
}

func (p *Pane) restartsSoFar() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.restarts
}

func TestEnvironment(t *testing.T) {
	p := start(t, Spec{Exec: `echo "$TERM $DOZER $DOZER_PANE"`}, 40, 3)
	waitFor(t, "exit", p.Dead)
	if got, _ := draw(p, 40, 3); !strings.Contains(got, "xterm-256color 1 1") {
		t.Errorf("env not set:\n%s", got)
	}
}

func TestArgvAndLabel(t *testing.T) {
	cases := []struct {
		spec  Spec
		argv  string
		label string
	}{
		{Spec{Shell: "/bin/zsh"}, "/bin/zsh -l", "zsh"},
		{Spec{Shell: "/bin/zsh", Exec: "htop"}, "/bin/zsh -lc htop", "htop"},
		{Spec{Shell: "/bin/zsh", Run: "ssh a"}, "/bin/zsh -ic ssh a; exec /bin/zsh", "ssh a"},
		{Spec{Shell: "/bin/zsh", Run: "ssh a", Title: "API"}, "/bin/zsh -ic ssh a; exec /bin/zsh", "API"},
	}
	for _, c := range cases {
		if got := strings.Join(c.spec.Argv(), " "); got != c.argv {
			t.Errorf("Argv(%+v) = %q, want %q", c.spec, got, c.argv)
		}
		if got := c.spec.Label(); got != c.label {
			t.Errorf("Label(%+v) = %q, want %q", c.spec, got, c.label)
		}
	}
}
