package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageRoundTripAndRun(t *testing.T) {
	dir := t.TempDir()
	src := "layout: \"2,1\"\ncwd: ~/work\nenv: {A: $HOME}\npanes: [{run: htop, cwd: sub}, {exec: 'echo \"$HOME\" `x`'}]\n"
	c, err := Parse([]byte(src), dir)
	if err != nil {
		t.Fatal(err)
	}
	script, err := c.Package("my-tool", "test", "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	y, ok := Unpack(script)
	if !ok {
		t.Fatal("Unpack failed")
	}
	// Written as authored, not as expanded on this machine.
	for _, want := range []string{"cwd: ~/work", "{A: $HOME}", "cwd: sub", "name: my-tool"} {
		if !strings.Contains(string(y), want) {
			t.Errorf("embedded YAML lacks %q:\n%s", want, y)
		}
	}
	path := filepath.Join(dir, "my-tool.sh")
	if err := os.WriteFile(path, script, 0o755); err != nil {
		t.Fatal(err)
	}

	// --show-config prints exactly the embedded YAML (no shell expansion).
	out, err := exec.Command("sh", path, "--show-config").Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(y) {
		t.Errorf("--show-config:\n%s\nwant:\n%s", out, y)
	}

	// Running it pipes the YAML to `dozer -c -` and passes flags on. Tests
	// have no terminal, so disable the script's terminal check for this run.
	fake := filepath.Join(dir, "fake-dozer")
	os.WriteFile(fake, []byte("#!/bin/sh\necho \"ARGS: $*\"\ncat\n"), 0o755)
	body := strings.Replace(string(script), "if [ ! -t 1 ]; then", "if false; then", 1)
	os.WriteFile(path, []byte(body), 0o755)
	cmd := exec.Command("sh", path, "--prefix", "C-b")
	cmd.Env = append(os.Environ(), "DOZER_BIN="+fake)
	out, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.HasPrefix(got, "ARGS: -c - --prefix C-b\n") {
		t.Errorf("args: %q", strings.SplitN(got, "\n", 2)[0])
	}
	if strings.SplitN(got, "\n", 2)[1] != string(y) {
		t.Errorf("stdin content:\n%s\nwant:\n%s", got, y)
	}
	// And it loads back to the same launch.
	c2, err := Parse(y, dir)
	if err != nil {
		t.Fatal(err)
	}
	c.Name, c.Source = "my-tool", ""
	if c2.Describe() != c.Describe() {
		t.Errorf("round trip differs:\n%s\nvs\n%s", c2.Describe(), c.Describe())
	}
}

func TestPackageNeedsTerminalAndDozer(t *testing.T) {
	c, _ := Parse([]byte(`layout: "1"`), "")
	script, _ := c.Package("t", "test", "today")
	path := filepath.Join(t.TempDir(), "t.sh")
	os.WriteFile(path, script, 0o755)
	cmd := exec.Command("sh", path)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "DOZER_BIN=/nonexistent/dozer"}
	out, err := cmd.CombinedOutput()
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 127 || !strings.Contains(string(out), "needs dozer") {
		t.Errorf("missing dozer: err=%v out=%q", err, out)
	}
	// A stand-in dozer that exists everywhere (macOS has no /bin/true).
	fake := filepath.Join(t.TempDir(), "dozer")
	os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	cmd = exec.Command("sh", path) // stdout is a pipe here, not a terminal
	cmd.Env = append(os.Environ(), "DOZER_BIN="+fake)
	out, _ = cmd.CombinedOutput()
	if !strings.Contains(string(out), "needs to run in a terminal") {
		t.Errorf("no-terminal check: %q", out)
	}
}

func TestPackageName(t *testing.T) {
	if n, err := PackageName("tools/my-tool.sh"); err != nil || n != "my-tool" {
		t.Errorf("got %q %v", n, err)
	}
	if _, err := PackageName("bad name.sh"); err == nil {
		t.Error("spaces should be rejected")
	}
}
