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
	c, err := Parse([]byte("layout: \"2,1\"\npanes: [{run: htop, cwd: sub}, {exec: 'echo \"$HOME\" `x`'}]\n"), dir)
	if err != nil {
		t.Fatal(err)
	}
	script, err := c.Package("my-tool", dir, "test", "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "my-tool.sh")
	if err := os.WriteFile(path, script, 0o755); err != nil {
		t.Fatal(err)
	}

	// dozer reads the package directly (Unpack), relative to its folder.
	c2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c2.Name != "my-tool" || c2.Panes[0].Dir != filepath.Join(dir, "sub") || c2.Panes[1].Exec != `echo "$HOME" `+"`x`" {
		t.Errorf("unpacked: name=%q dir=%q exec=%q", c2.Name, c2.Panes[0].Dir, c2.Panes[1].Exec)
	}

	// --show-config prints exactly the embedded YAML (no shell expansion).
	out, err := exec.Command("sh", path, "--show-config").Output()
	if err != nil {
		t.Fatal(err)
	}
	y, _ := Unpack(script)
	if string(out) != string(y) {
		t.Errorf("--show-config:\n%s\nwant:\n%s", out, y)
	}

	// Running it execs dozer with the config on fd 3 and passes flags on.
	fake := filepath.Join(dir, "fake-dozer")
	os.WriteFile(fake, []byte("#!/bin/sh\necho \"ARGS: $*\"\ncat /dev/fd/3\n"), 0o755)
	cmd := exec.Command("sh", path, "--prefix", "C-b")
	cmd.Env = append(os.Environ(), "DOZER_BIN="+fake)
	out, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.HasPrefix(got, "ARGS: --base "+dir+" -c /dev/fd/3 --prefix C-b\n") {
		t.Errorf("args: %q", strings.SplitN(got, "\n", 2)[0])
	}
	if !strings.Contains(got, strings.TrimSpace(string(y))) {
		t.Errorf("fd 3 content:\n%s", got)
	}
	c3, err := Parse([]byte(got[strings.Index(got, "\n")+1:]), dir)
	if err != nil {
		t.Fatal(err)
	}
	c3.Source = c2.Source
	if c3.Describe() != c2.Describe() {
		t.Errorf("config via fd 3 differs:\n%s\nvs\n%s", c3.Describe(), c2.Describe())
	}
}

func TestPortablePath(t *testing.T) {
	cases := map[string]string{
		"/pkg/dir":       ".",
		"/pkg/dir/sub/x": "./sub/x",
		"/home/u":        "~",
		"/home/u/work":   "~/work",
		"/etc/other":     "/etc/other",
		"relative/stays": "relative/stays",
		"":               "",
	}
	for in, want := range cases {
		if got := portablePath(in, "/pkg/dir", "/home/u"); got != want {
			t.Errorf("portablePath(%q) = %q, want %q", in, got, want)
		}
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
