package config

import (
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
	c, err := Parse([]byte("mouse: true\npanes: [\"git status\", {exec: top, group: hosts}]\nprefix: C-b\nenv: {A: x}"), "/base")
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
		t.Errorf("warnings = %v, want mouse + group", c.Warnings)
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
