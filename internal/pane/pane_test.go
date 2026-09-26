package pane

import (
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/tomj59/dozer/internal/emu"
)

func startExec(t *testing.T, cmd string, cols, rows int) *Pane {
	t.Helper()
	em, err := emu.New("charm", cols, rows)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Start(1, Spec{Shell: "/bin/sh", Exec: cmd}, em, cols, rows, func() {})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	select {
	case <-p.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("pane did not exit")
	}
	return p
}

func screenText(p *Pane, cols, rows int) string {
	buf := uv.NewScreenBuffer(cols, rows)
	p.Draw(buf, uv.Rect(0, 0, cols, rows))
	return buf.String()
}

func TestExecOutputAndExitCode(t *testing.T) {
	p := startExec(t, `printf 'hello \033[31mred\033[0m'; echo; printf 'size:'; stty size; exit 7`, 40, 5)
	got := screenText(p, 40, 5)
	if !strings.Contains(got, "hello red") {
		t.Errorf("screen missing output:\n%s", got)
	}
	if !strings.Contains(got, "size:5 40") {
		t.Errorf("pty size not propagated:\n%s", got)
	}
	if p.ExitCode() != 7 {
		t.Errorf("exit code = %d, want 7", p.ExitCode())
	}
}

func TestEnvironment(t *testing.T) {
	p := startExec(t, `echo "$TERM $DOZER $DOZER_PANE"`, 40, 3)
	if got := screenText(p, 40, 3); !strings.Contains(got, "xterm-256color 1 1") {
		t.Errorf("env not set:\n%s", got)
	}
}

func TestArgv(t *testing.T) {
	cases := []struct {
		spec Spec
		want string
	}{
		{Spec{Shell: "/bin/zsh"}, "/bin/zsh -l"},
		{Spec{Shell: "/bin/zsh", Exec: "htop"}, "/bin/zsh -lc htop"},
		{Spec{Shell: "/bin/zsh", Run: "ssh a"}, "/bin/zsh -ic ssh a; exec /bin/zsh"},
	}
	for _, c := range cases {
		if got := strings.Join(c.spec.Argv(), " "); got != c.want {
			t.Errorf("Argv(%+v) = %q, want %q", c.spec, got, c.want)
		}
	}
}
