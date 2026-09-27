package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tomj59/dozer/internal/layout"
)

// Every shipped example must load cleanly.
func TestExamplesLoad(t *testing.T) {
	files, _ := filepath.Glob("../../examples/*.yaml")
	if len(files) == 0 {
		t.Fatal("no examples found")
	}
	want := map[string]int{"reference": 3, "dev": 3, "monitor": 5, "tree": 3, "lifecycle": 5, "ssh": 3}
	for _, f := range files {
		c, err := Load(f)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		name := strings.TrimSuffix(filepath.Base(f), ".yaml")
		if n := len(c.Panes); n != want[name] {
			t.Errorf("%s: %d panes, want %d", name, n, want[name])
		}
		if len(c.Warnings) > 0 {
			t.Errorf("%s: unexpected warnings %v", name, c.Warnings)
		}
	}
}

func TestTreeForm(t *testing.T) {
	c, err := Load("../../examples/tree.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if c.Panes[0].Run != "ls -la" || c.Panes[1].Restart != "on-failure" || c.Panes[2].Title != "shell" {
		t.Errorf("panes = %+v", c.Panes)
	}
	r := layout.Solve(c.Layout, 101, 40, c.MinPane) // 100 cells + 1 divider
	if r.Slots[0].W != 60 || r.Slots[2].H != 12 {
		t.Errorf("slots = %v", r.Slots)
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"layout: \"2,x\"":                             "bad layout",
		"bogus: 1":                                    "field bogus not found",
		"panes: [a, b, c, d]":                         "4 panes listed but the layout has 3",
		"panes: [{run: a, exec: b}]":                  "either run: or exec:",
		"panes: [{run: a, restart: always}]":          "only applies to exec:",
		"panes: [{exec: a, restart: sometimes}]":      "restart must be",
		"panes: {api: {run: x}}":                      "named panes need a tree layout",
		"layout: {cols: [{pane: a}, {pane: a}]}":      "used twice",
		"layout: {cols: [{pane: a}]}\npanes: {b: {}}": "pane \"b\" is not in the layout",
		"heights: [10, 20, 30]":                       "layout has 2 rows",
		"prefix: F1":                                  "use C-a",
		"version: 2":                                  "unsupported version",
		"panes: [{run: a, colour: red}]":              "field colour not found",
	}
	for in, want := range cases {
		_, err := Parse([]byte(in), "")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) error = %v, want it to mention %q", in, err, want)
		}
	}
}

func TestWarningsAndShorthands(t *testing.T) {
	c, err := Parse([]byte("multi_input: true\npanes: [\"git status\", {exec: top, group: hosts}]\nprefix: C-b\nenv: {A: x}"), "/base")
	if err != nil {
		t.Fatal(err)
	}
	if c.Prefix != 2 {
		t.Errorf("prefix = %d, want 2 (C-b)", c.Prefix)
	}
	if c.Panes[0].Run != "git status" || c.Panes[1].Exec != "top" {
		t.Errorf("panes = %+v", c.Panes)
	}
	if len(c.Panes[2].Env) != 1 || c.Panes[2].Env[0] != "A=x" {
		t.Errorf("default env not inherited: %+v", c.Panes[2])
	}
	if len(c.Warnings) != 2 {
		t.Errorf("warnings = %v, want multi_input + group", c.Warnings)
	}
}

func TestExpandDir(t *testing.T) {
	if got := expandDir("sub", "/base"); got != "/base/sub" {
		t.Errorf("relative dir = %q", got)
	}
	if got := expandDir("/abs", "/base"); got != "/abs" {
		t.Errorf("absolute dir = %q", got)
	}
}

// --save output must load back to the same launch.
func TestSaveRoundTrip(t *testing.T) {
	files, _ := filepath.Glob("../../examples/*.yaml")
	for _, f := range files {
		c, err := Load(f)
		if err != nil {
			t.Fatal(err)
		}
		out, err := c.YAML("saved by test")
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		c2, err := Parse(out, "")
		if err != nil {
			t.Fatalf("%s: saved YAML does not load: %v\n%s", f, err, out)
		}
		c.Source, c2.Source = "", ""
		a, b := c.Describe(), c2.Describe()
		if a != b {
			t.Errorf("%s: round trip differs\n--- original\n%s--- saved\n%s--- yaml\n%s", f, a, b, out)
		}
	}
}

func TestSaveAfterResize(t *testing.T) {
	c, _ := Parse([]byte(`layout: "2,1"`), "")
	r := layout.Solve(c.Layout, 101, 40, c.MinPane)
	layout.MoveBorder(c.Layout, 0, 10, 0, r, c.MinPane)
	out, _ := c.YAML("")
	if !strings.Contains(string(out), "widths: {1: [60fr, 40fr]}") {
		t.Errorf("resized widths not saved:\n%s", out)
	}
	c2, err := Parse(out, "")
	if err != nil {
		t.Fatal(err)
	}
	if s := layout.Solve(c2.Layout, 101, 40, c2.MinPane).Slots; s[0].W != 60 {
		t.Errorf("reloaded width %d, want 60", s[0].W)
	}
}

// Every packaged manual test case must load cleanly, be named after its
// file, and carry a description (the guide pane shows it).
func TestManualCases(t *testing.T) {
	files, _ := filepath.Glob("../../tests/cases/*.sh")
	if len(files) == 0 {
		t.Fatal("no test cases found")
	}
	for _, f := range files {
		c, err := Load(f)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		if want := strings.TrimSuffix(filepath.Base(f), ".sh"); c.Name != want {
			t.Errorf("%s: name %q, want %q", f, c.Name, want)
		}
		if !strings.HasPrefix(c.Description, strings.SplitN(c.Name, "-", 2)[0]+": ") {
			t.Errorf("%s: description should start with its ID", f)
		}
		if len(c.Warnings) > 0 {
			t.Errorf("%s: warnings %v", f, c.Warnings)
		}
	}
}

func TestMaxRestarts(t *testing.T) {
	c, err := Parse([]byte("panes: [{exec: x, restart: always, max_restarts: 3}, {exec: y, restart: on-failure, max_restarts: unlimited}, {exec: z, restart: always}]"), "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Panes[0].MaxRestarts != 3 || c.Panes[1].MaxRestarts != -1 || c.Panes[2].MaxRestarts != 0 {
		t.Errorf("max restarts = %d %d %d", c.Panes[0].MaxRestarts, c.Panes[1].MaxRestarts, c.Panes[2].MaxRestarts)
	}
	if d := c.Describe(); !strings.Contains(d, "max 3") || !strings.Contains(d, "max unlimited") || !strings.Contains(d, "max 5") {
		t.Errorf("describe:\n%s", d)
	}
	out, _ := c.YAML("")
	c2, err := Parse(out, "")
	if err != nil || c2.Describe() != c.Describe() {
		t.Errorf("round trip: %v\n%s", err, out)
	}
	for in, want := range map[string]string{
		"panes: [{exec: x, max_restarts: 3}]":                     "needs restart:",
		"panes: [{exec: x, restart: always, max_restarts: 0}]":    "must be a number",
		"panes: [{exec: x, restart: always, max_restarts: lots}]": "must be a number",
	} {
		if _, err := Parse([]byte(in), ""); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) = %v, want %q", in, err, want)
		}
	}
}

// A bare ~ is YAML null; for cwd it must still mean the home folder, and
// --save must write strings so they read back as strings.
func TestTildeCwd(t *testing.T) {
	home, _ := os.UserHomeDir()
	c, err := Parse([]byte("cwd: ~\npanes: [{cwd: ~}, {cwd: null}, {cwd: \"~/x\"}]"), "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Panes[0].Dir != home || c.Panes[2].Dir != filepath.Join(home, "x") {
		t.Errorf("dirs = %q %q %q", c.Panes[0].Dir, c.Panes[1].Dir, c.Panes[2].Dir)
	}
	if c.Panes[1].Dir != home { // null = unset → inherits the top-level ~
		t.Errorf("null cwd should inherit, got %q", c.Panes[1].Dir)
	}
	out, _ := c.YAML("")
	c2, err := Parse(out, "")
	if err != nil || c2.Panes[0].Dir != home {
		t.Errorf("round trip: %v %q\n%s", err, c2.Panes[0].Dir, out)
	}
	for _, v := range []string{"true", "123", "null"} {
		c, _ := Parse([]byte(`panes: [{run: "`+v+`"}]`), "")
		out, _ := c.YAML("")
		if c2, err := Parse(out, ""); err != nil || c2.Panes[0].Run != v {
			t.Errorf("run %q did not survive --save: %v\n%s", v, err, out)
		}
	}
}

func TestMouseAndScrollback(t *testing.T) {
	c, err := Parse([]byte("mouse: false\nscrollback: 500\n"), "/base")
	if err != nil {
		t.Fatal(err)
	}
	if c.Mouse || c.Scrollback != 500 || len(c.Warnings) != 0 {
		t.Fatalf("mouse=%v scrollback=%d warnings=%v", c.Mouse, c.Scrollback, c.Warnings)
	}
	if d := Defaults(); !d.Mouse || d.Scrollback != DefaultScrollback {
		t.Fatalf("defaults: %+v", d)
	}
	y, err := c.YAML("")
	if err != nil || !strings.Contains(string(y), "mouse: false") || !strings.Contains(string(y), "scrollback: 500") {
		t.Fatalf("save: %v\n%s", err, y)
	}
	if _, err := Parse([]byte("scrollback: -1"), "/"); err == nil {
		t.Fatal("negative scrollback accepted")
	}
}
