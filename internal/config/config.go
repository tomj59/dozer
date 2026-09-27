// Package config loads dozer's YAML launch configuration (docs/SPEC.md §5)
// and merges command-line overrides into it.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/tomj59/dozer/internal/layout"
	"github.com/tomj59/dozer/internal/pane"
)

// Config is a resolved launch configuration: everything dozer needs to
// build the session.
type Config struct {
	Name              string
	Description       string // free text: what this workspace (or test case) is for
	Source            string // file it came from, "" if flags only
	Layout            *layout.Node
	Panes             []pane.Spec // indexed by pane number (reading order)
	PaneNames         []string    // tree-form layouts: the pane names, by number
	Prefix            byte
	MinPane           layout.Min
	QuitWhenAllExited bool
	StatusBar         string   // "bottom" (default), "top" or "off"
	Warnings          []string // accepted but not implemented yet, etc.
}

// file mirrors the YAML schema.
type file struct {
	Version           int                 `yaml:"version"`
	Name              string              `yaml:"name"`
	Description       string              `yaml:"description"`
	Shell             string              `yaml:"shell"`
	Cwd               yaml.Node           `yaml:"cwd"`
	Env               map[string]string   `yaml:"env"`
	Prefix            string              `yaml:"prefix"`
	Layout            yaml.Node           `yaml:"layout"`
	Heights           []string            `yaml:"heights"`
	Widths            map[string][]string `yaml:"widths"` // row number → sizes
	Panes             yaml.Node           `yaml:"panes"`
	MinPane           *struct{ W, H int } `yaml:"min_pane"`
	QuitWhenAllExited bool                `yaml:"quit_when_all_exited"`

	// Accepted now, implemented in later milestones.
	Mouse      yaml.Node `yaml:"mouse"`
	Scrollback yaml.Node `yaml:"scrollback"`
	StatusBar  string    `yaml:"status_bar"`
	MultiInput yaml.Node `yaml:"multi_input"`
	Keys       yaml.Node `yaml:"keys"`
}

type paneFile struct {
	Pane     string            `yaml:"pane"` // name (list form, optional)
	Title    string            `yaml:"title"`
	Run      string            `yaml:"run"`
	Exec     string            `yaml:"exec"`
	Cwd      yaml.Node         `yaml:"cwd"`
	Env      map[string]string `yaml:"env"`
	Shell    string            `yaml:"shell"`
	Restart  string            `yaml:"restart"`
	MaxRest  string            `yaml:"max_restarts"` // a number, or "unlimited"
	Readonly yaml.Node         `yaml:"readonly"`
	Group    yaml.Node         `yaml:"group"`
	Min      yaml.Node         `yaml:"min"`
}

// pathOf reads a cwd value as written. A bare ~ is YAML for null, so
// `cwd: ~` would silently mean "no cwd"; here it keeps meaning the home
// folder. An explicit `null` (or no key) means unset.
func pathOf(n *yaml.Node) string {
	switch {
	case n.Kind == 0:
		return ""
	case n.Tag == "!!null":
		if n.Value == "~" {
			return "~"
		}
		return ""
	}
	return n.Value
}

// Defaults returns the configuration for a bare `dozer`.
func Defaults() *Config {
	root, _ := layout.Parse("default")
	return &Config{Layout: root, Panes: make([]pane.Spec, root.Panes()), Prefix: 0x01, MinPane: layout.Min{W: 20, H: 5}, StatusBar: "bottom"}
}

// Load reads and resolves a YAML config file, or the config embedded in a
// packaged dozer script (see Package). Relative cwd paths resolve against
// the file's folder. For config text with no file (stdin, --config-text),
// use Parse with baseDir "": relative paths then resolve against the
// directory dozer runs in.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if y, ok := Unpack(data); ok {
		data = y
	}
	c, err := Parse(data, filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	c.Source = path
	return c, nil
}

// Parse resolves YAML config data. baseDir resolves relative cwd paths.
func Parse(data []byte, baseDir string) (*Config, error) {
	var f file
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil && !errors.Is(err, io.EOF) { // empty file = defaults
		return nil, cleanYAMLError(err)
	}
	if f.Version != 0 && f.Version != 1 {
		return nil, fmt.Errorf("unsupported version %d (this dozer reads version 1)", f.Version)
	}
	c := Defaults()
	c.Name = f.Name
	c.Description = strings.TrimRight(f.Description, "\n")

	for key, n := range map[string]*yaml.Node{"mouse": &f.Mouse, "scrollback": &f.Scrollback,
		"multi_input": &f.MultiInput, "keys": &f.Keys} {
		if n.Kind != 0 {
			c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: %q is not implemented yet; ignored", n.Line, key))
		}
	}
	sort.Strings(c.Warnings)

	if f.Prefix != "" {
		b, err := ParsePrefix(f.Prefix)
		if err != nil {
			return nil, err
		}
		c.Prefix = b
	}
	if f.MinPane != nil {
		if f.MinPane.W < 1 || f.MinPane.H < 1 {
			return nil, fmt.Errorf("min_pane: w and h must be at least 1")
		}
		c.MinPane = layout.Min{W: f.MinPane.W, H: f.MinPane.H}
	}
	c.QuitWhenAllExited = f.QuitWhenAllExited
	switch f.StatusBar {
	case "":
	case "top", "bottom", "off":
		c.StatusBar = f.StatusBar
	default:
		return nil, fmt.Errorf("status_bar must be top, bottom or off")
	}

	// Layout: shorthand string or a rows/cols tree.
	names := map[string]int{} // tree form: pane name → index
	switch f.Layout.Kind {
	case 0:
		// default layout
	case yaml.ScalarNode:
		root, err := layout.Parse(f.Layout.Value)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", f.Layout.Line, err)
		}
		c.Layout = root
	case yaml.MappingNode:
		root, err := parseTree(&f.Layout, names)
		if err != nil {
			return nil, err
		}
		c.Layout = root
	default:
		return nil, fmt.Errorf("line %d: layout must be a shorthand like \"2,1\" or a rows/cols tree", f.Layout.Line)
	}
	if len(f.Heights) > 0 || len(f.Widths) > 0 {
		if len(names) > 0 {
			return nil, errors.New("heights/widths apply to shorthand layouts; put size: on tree nodes instead")
		}
		if err := applySizes(c.Layout, f.Heights, f.Widths); err != nil {
			return nil, err
		}
	}
	c.Panes = make([]pane.Spec, c.Layout.Panes())
	if len(names) > 0 {
		c.PaneNames = make([]string, len(c.Panes))
		for name, i := range names {
			c.PaneNames[i] = name
		}
	}

	// Panes: a list in reading order, or a map keyed by tree pane names.
	base := pane.Spec{Shell: f.Shell, Dir: expandDir(pathOf(&f.Cwd), baseDir), Env: envList(f.Env),
		RawDir: pathOf(&f.Cwd), RawEnv: rawEnvList(f.Env)}
	for i := range c.Panes {
		c.Panes[i] = base
	}
	switch f.Panes.Kind {
	case 0:
	case yaml.SequenceNode:
		if len(f.Panes.Content) > len(c.Panes) {
			return nil, fmt.Errorf("line %d: %d panes listed but the layout has %d", f.Panes.Line, len(f.Panes.Content), len(c.Panes))
		}
		for i, n := range f.Panes.Content {
			spec, warn, err := decodePane(n, base, baseDir)
			if err != nil {
				return nil, err
			}
			c.Warnings = append(c.Warnings, warn...)
			c.Panes[i] = spec
		}
	case yaml.MappingNode:
		if len(names) == 0 {
			return nil, fmt.Errorf("line %d: named panes need a tree layout with `pane: <name>` leaves; use a list for shorthand layouts", f.Panes.Line)
		}
		for i := 0; i < len(f.Panes.Content); i += 2 {
			key, val := f.Panes.Content[i], f.Panes.Content[i+1]
			idx, ok := names[key.Value]
			if !ok {
				return nil, fmt.Errorf("line %d: pane %q is not in the layout", key.Line, key.Value)
			}
			spec, warn, err := decodePane(val, base, baseDir)
			if err != nil {
				return nil, err
			}
			c.Warnings = append(c.Warnings, warn...)
			if spec.Title == "" {
				spec.Title = key.Value
			}
			c.Panes[idx] = spec
		}
	default:
		return nil, fmt.Errorf("line %d: panes must be a list or a map", f.Panes.Line)
	}
	return c, nil
}

func cleanYAMLError(err error) error {
	msg := strings.TrimPrefix(err.Error(), "yaml: ")
	msg = strings.ReplaceAll(msg, "unmarshal errors:\n  ", "")
	msg = strings.ReplaceAll(msg, " in type config.file", "")
	msg = strings.ReplaceAll(msg, " in type config.paneFile", "")
	return errors.New(msg)
}

func decodePane(n *yaml.Node, base pane.Spec, baseDir string) (pane.Spec, []string, error) {
	var pf paneFile
	if n.Kind == yaml.ScalarNode { // shorthand: a bare string is a run: command
		pf.Run = n.Value
	} else if err := decodeStrict(n, &pf); err != nil {
		return pane.Spec{}, nil, err
	}
	if pf.Run != "" && pf.Exec != "" {
		return pane.Spec{}, nil, fmt.Errorf("line %d: a pane has either run: or exec:, not both", n.Line)
	}
	switch pf.Restart {
	case "", pane.RestartNever, pane.RestartOnFailure, pane.RestartAlways:
	default:
		return pane.Spec{}, nil, fmt.Errorf("line %d: restart must be never, on-failure or always", n.Line)
	}
	if pf.Restart != "" && pf.Restart != pane.RestartNever && pf.Exec == "" {
		return pane.Spec{}, nil, fmt.Errorf("line %d: restart: only applies to exec: panes (a run: pane keeps its shell)", n.Line)
	}
	maxRestarts := 0
	switch pf.MaxRest {
	case "":
	case "unlimited":
		maxRestarts = -1
	default:
		v, err := strconv.Atoi(pf.MaxRest)
		if err != nil || v < 1 {
			return pane.Spec{}, nil, fmt.Errorf("line %d: max_restarts must be a number ≥ 1 or \"unlimited\"", n.Line)
		}
		maxRestarts = v
	}
	if pf.MaxRest != "" && (pf.Restart == "" || pf.Restart == pane.RestartNever) {
		return pane.Spec{}, nil, fmt.Errorf("line %d: max_restarts needs restart: on-failure or always", n.Line)
	}
	var warn []string
	for key, v := range map[string]*yaml.Node{"readonly": &pf.Readonly, "group": &pf.Group, "min": &pf.Min} {
		if v.Kind != 0 {
			warn = append(warn, fmt.Sprintf("line %d: pane %q is not implemented yet; ignored", v.Line, key))
		}
	}
	sort.Strings(warn)
	s := base
	s.Title, s.Run, s.Exec, s.Restart, s.MaxRestarts = pf.Title, pf.Run, pf.Exec, pf.Restart, maxRestarts
	if pf.Shell != "" {
		s.Shell = pf.Shell
	}
	if cwd := pathOf(&pf.Cwd); cwd != "" {
		s.Dir, s.RawDir = expandDir(cwd, baseDir), cwd
	}
	if len(pf.Env) > 0 {
		s.Env = append(append([]string(nil), base.Env...), envList(pf.Env)...)
		s.RawEnv = append(append([]string(nil), base.RawEnv...), rawEnvList(pf.Env)...)
	}
	return s, warn, nil
}

func decodeStrict(n *yaml.Node, v any) error {
	// yaml.Node.Decode doesn't support KnownFields; round-trip instead.
	b, err := yaml.Marshal(n)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		// Line numbers from the round-trip are relative; report the node's.
		msg := cleanYAMLError(err).Error()
		if i := strings.Index(msg, ": "); strings.HasPrefix(msg, "line ") && i > 0 {
			msg = msg[i+2:]
		}
		return fmt.Errorf("line %d: %s", n.Line, msg)
	}
	return nil
}

// parseTree reads a {rows|cols: [...], size:} / {pane: name, size:} tree.
func parseTree(n *yaml.Node, names map[string]int) (*layout.Node, error) {
	if n.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("line %d: expected a layout node ({rows: …}, {cols: …} or {pane: …})", n.Line)
	}
	out := &layout.Node{Pane: -1}
	var kids *yaml.Node
	seen := ""
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		switch k.Value {
		case "size":
			sz, err := layout.ParseSize(v.Value)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", v.Line, err)
			}
			out.Size = sz
		case "rows", "cols", "pane":
			if seen != "" {
				return nil, fmt.Errorf("line %d: a layout node has one of rows, cols or pane (found %s and %s)", k.Line, seen, k.Value)
			}
			seen = k.Value
			if k.Value == "pane" {
				if _, dup := names[v.Value]; dup {
					return nil, fmt.Errorf("line %d: pane name %q used twice", v.Line, v.Value)
				}
				if v.Value == "" {
					return nil, fmt.Errorf("line %d: pane needs a name", v.Line)
				}
				out.Pane = len(names)
				names[v.Value] = out.Pane
			} else {
				out.Dir = layout.Rows
				if k.Value == "cols" {
					out.Dir = layout.Cols
				}
				kids = v
			}
		default:
			return nil, fmt.Errorf("line %d: unknown layout key %q (use rows, cols, pane, size)", k.Line, k.Value)
		}
	}
	switch {
	case seen == "":
		return nil, fmt.Errorf("line %d: layout node needs rows, cols or pane", n.Line)
	case kids != nil:
		if kids.Kind != yaml.SequenceNode || len(kids.Content) == 0 {
			return nil, fmt.Errorf("line %d: %s must be a non-empty list", kids.Line, seen)
		}
		for _, ch := range kids.Content {
			node, err := parseTree(ch, names)
			if err != nil {
				return nil, err
			}
			out.Children = append(out.Children, node)
		}
	}
	if len(names) > 9 {
		return nil, fmt.Errorf("line %d: the layout has more than 9 panes", n.Line)
	}
	return out, nil
}

func applySizes(root *layout.Node, heights []string, widths map[string][]string) error {
	if len(heights) > 0 {
		sz, err := parseList(heights)
		if err != nil {
			return fmt.Errorf("heights: %w", err)
		}
		if err := layout.SetHeights(root, sz); err != nil {
			return err
		}
	}
	for key, list := range widths {
		row, err := strconv.Atoi(key)
		if err != nil {
			return fmt.Errorf("widths: row %q is not a number", key)
		}
		sz, err := parseList(list)
		if err != nil {
			return fmt.Errorf("widths: %w", err)
		}
		if err := layout.SetWidths(root, row, sz); err != nil {
			return err
		}
	}
	return nil
}

func parseList(list []string) ([]layout.Size, error) {
	var out []layout.Size
	for _, s := range list {
		sz, err := layout.ParseSize(s)
		if err != nil {
			return nil, err
		}
		out = append(out, sz)
	}
	return out, nil
}

func rawEnvList(m map[string]string) []string {
	var out []string
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

func envList(m map[string]string) []string {
	var out []string
	for k, v := range m {
		out = append(out, k+"="+os.ExpandEnv(v))
	}
	sort.Strings(out)
	return out
}

// expandDir expands ~ and $VARS, and resolves relative paths against the
// config file's directory.
func expandDir(dir, baseDir string) string {
	if dir == "" {
		return ""
	}
	dir = os.ExpandEnv(dir)
	if dir == "~" || strings.HasPrefix(dir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
		}
	}
	if !filepath.IsAbs(dir) && baseDir != "" {
		dir = filepath.Join(baseDir, dir)
	}
	return dir
}

// ParsePrefix parses "C-a", "C-b", "^a" … into a control byte.
func ParsePrefix(s string) (byte, error) {
	t := strings.ToLower(strings.TrimSpace(s))
	for _, p := range []string{"c-", "ctrl-", "ctrl+", "^"} {
		if strings.HasPrefix(t, p) && len(t) == len(p)+1 {
			ch := t[len(p)]
			if ch >= 'a' && ch <= 'z' {
				return ch - 'a' + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("prefix %q: use C-a … C-z", s)
}

// Find resolves what to load for `dozer [name|file]`:
//   - an existing file path is loaded as-is
//   - a name loads ~/.config/dozer/<name>.yaml (or .yml)
//   - no argument loads ./.dozer.yaml, then ~/.config/dozer/config.yaml,
//     if present; otherwise "" (built-in defaults).
func Find(arg string) (string, error) {
	dir := configDir()
	if arg != "" {
		if st, err := os.Stat(arg); err == nil && !st.IsDir() {
			return arg, nil
		}
		for _, ext := range []string{".yaml", ".yml"} {
			p := filepath.Join(dir, arg+ext)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
		return "", fmt.Errorf("no config file %q and no profile %s", arg, filepath.Join(dir, arg+".yaml"))
	}
	for _, p := range []string{".dozer.yaml", filepath.Join(dir, "config.yaml")} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", nil
}

func configDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "dozer")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dozer")
}

// Describe returns a human summary of the resolved config (for --check).
func (c *Config) Describe() string {
	var b strings.Builder
	src := c.Source
	if src == "" {
		src = "(flags/defaults)"
	}
	fmt.Fprintf(&b, "config: %s\n", src)
	if c.Name != "" {
		fmt.Fprintf(&b, "name:   %s\n", c.Name)
	}
	if c.Description != "" {
		for _, line := range strings.Split(c.Description, "\n") {
			fmt.Fprintf(&b, "  | %s\n", line)
		}
	}
	fmt.Fprintf(&b, "prefix: C-%c   min pane: %dx%d   status bar: %s   quit when all exited: %v\n", 'a'+c.Prefix-1, c.MinPane.W, c.MinPane.H, c.StatusBar, c.QuitWhenAllExited)
	fmt.Fprintf(&b, "layout: %s\n", describeNode(c.Layout))
	for i, p := range c.Panes {
		what := "shell"
		switch {
		case p.Exec != "":
			what = "exec: " + p.Exec
		case p.Run != "":
			what = "run:  " + p.Run
		}
		fmt.Fprintf(&b, "  [%d] %-14s %s", i+1, p.Label(), what)
		if p.Dir != "" {
			fmt.Fprintf(&b, "   (cwd %s)", p.Dir)
		}
		if p.Restart != "" && p.Restart != pane.RestartNever {
			limit := strconv.Itoa(p.MaxRestarts)
			switch {
			case p.MaxRestarts == 0:
				limit = strconv.Itoa(pane.DefaultMaxRestarts)
			case p.MaxRestarts < 0:
				limit = "unlimited"
			}
			fmt.Fprintf(&b, "   (restart %s, max %s)", p.Restart, limit)
		}
		b.WriteByte('\n')
	}
	for _, w := range c.Warnings {
		fmt.Fprintf(&b, "warning: %s\n", w)
	}
	return b.String()
}

func describeNode(n *layout.Node) string {
	size := ""
	if n.Size.Unit != layout.Auto {
		size = "@" + n.Size.String()
	}
	if n.Leaf() {
		return fmt.Sprintf("[%d]%s", n.Pane+1, size)
	}
	var parts []string
	for _, ch := range n.Children {
		parts = append(parts, describeNode(ch))
	}
	dir := "rows"
	if n.Dir == layout.Cols {
		dir = "cols"
	}
	return fmt.Sprintf("%s(%s)%s", dir, strings.Join(parts, " "), size)
}
