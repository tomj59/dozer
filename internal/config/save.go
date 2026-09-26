package config

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/tomj59/dozer/internal/layout"
	"github.com/tomj59/dozer/internal/pane"
)

// YAML renders the resolved configuration as a canonical config file that
// Parse reads back to the same launch (docs/SPEC.md §4.9). It saves
// configuration only: layout, sizes, display prefs and each pane's launch
// command, never process state.
//
// Shorthand-shaped layouts are written as "2,1" plus heights/widths; any
// other tree is written in tree form, keeping the config's pane names
// (or p1, p2, … for trees built another way).
func (c *Config) YAML(header string) ([]byte, error) {
	doc := &yaml.Node{Kind: yaml.MappingNode}
	add := func(key string, v *yaml.Node) {
		doc.Content = append(doc.Content, str(key), v)
	}
	add("version", str("1"))
	if c.Name != "" {
		add("name", str(c.Name))
	}
	def := Defaults()
	if c.Prefix != def.Prefix {
		add("prefix", str(fmt.Sprintf("C-%c", 'a'+c.Prefix-1)))
	}
	if c.MinPane != def.MinPane {
		add("min_pane", flow(str("w"), str(strconv.Itoa(c.MinPane.W)), str("h"), str(strconv.Itoa(c.MinPane.H))))
	}
	if c.StatusBar != "" && c.StatusBar != def.StatusBar {
		add("status_bar", str(c.StatusBar))
	}
	if c.QuitWhenAllExited {
		add("quit_when_all_exited", str("true"))
	}

	// Settings every pane shares are written once at the top.
	shared := pane.Spec{}
	if len(c.Panes) > 0 {
		shared = c.Panes[0]
		for _, p := range c.Panes[1:] {
			if p.Shell != shared.Shell {
				shared.Shell = ""
			}
			if p.Dir != shared.Dir {
				shared.Dir = ""
			}
			if strings.Join(p.Env, "\x00") != strings.Join(shared.Env, "\x00") {
				shared.Env = nil
			}
		}
		if shared.Shell != "" {
			add("shell", str(shared.Shell))
		}
		if shared.Dir != "" {
			add("cwd", str(shared.Dir))
		}
		if len(shared.Env) > 0 {
			add("env", envNode(shared.Env))
		}
	}

	if rows, ok := shorthand(c.Layout); ok {
		var counts, heights []string
		var widths []*yaml.Node
		anyH := false
		for i, r := range rows {
			counts = append(counts, strconv.Itoa(len(r.cols)))
			heights = append(heights, sizeText(r.size))
			anyH = anyH || r.size.Unit != layout.Auto
			var ws []string
			anyW := false
			for _, col := range r.cols {
				ws = append(ws, sizeText(col.Size))
				anyW = anyW || col.Size.Unit != layout.Auto
			}
			if anyW && len(r.cols) > 1 {
				widths = append(widths, str(strconv.Itoa(i+1)), flowList(ws))
			}
		}
		add("layout", quoted(strings.Join(counts, ",")))
		if anyH && len(rows) > 1 {
			add("heights", flowList(heights))
		}
		if len(widths) > 0 {
			add("widths", &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle, Content: widths})
		}
		list := &yaml.Node{Kind: yaml.SequenceNode}
		for _, p := range c.Panes {
			list.Content = append(list.Content, paneNode(p, shared))
		}
		add("panes", list)
	} else {
		add("layout", treeNode(c.Layout, c.paneName))
		m := &yaml.Node{Kind: yaml.MappingNode}
		for i, p := range c.Panes {
			name := c.paneName(i)
			if p.Title == name {
				p.Title = "" // the name doubles as the title
			}
			m.Content = append(m.Content, str(name), paneNode(p, shared))
		}
		add("panes", m)
	}

	var buf bytes.Buffer
	for _, line := range strings.Split(strings.TrimSpace(header), "\n") {
		if line != "" {
			buf.WriteString("# " + line + "\n")
		}
	}
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{doc}}); err != nil {
		return nil, err
	}
	return buf.Bytes(), enc.Close()
}

type row struct {
	size layout.Size
	cols []*layout.Node
}

// shorthand reports whether n has the shape Parse builds from "2,1" etc.
func shorthand(n *layout.Node) ([]row, bool) {
	isRow := func(r *layout.Node) ([]*layout.Node, bool) {
		if r.Leaf() {
			return []*layout.Node{r}, true
		}
		if r.Dir != layout.Cols {
			return nil, false
		}
		for _, ch := range r.Children {
			if !ch.Leaf() {
				return nil, false
			}
		}
		return r.Children, true
	}
	if n.Leaf() || n.Dir == layout.Cols {
		cols, ok := isRow(n)
		return []row{{cols: cols}}, ok
	}
	var rows []row
	for _, r := range n.Children {
		cols, ok := isRow(r)
		if !ok {
			return nil, false
		}
		rows = append(rows, row{size: r.Size, cols: cols})
	}
	// Leaves must be numbered in reading order for the list form.
	i := 0
	for _, r := range rows {
		for _, c := range r.cols {
			if c.Pane != i {
				return nil, false
			}
			i++
		}
	}
	return rows, true
}

// paneName is pane i's name in tree form: its config name, or p1, p2, …
func (c *Config) paneName(i int) string {
	if i < len(c.PaneNames) && c.PaneNames[i] != "" {
		return c.PaneNames[i]
	}
	return fmt.Sprintf("p%d", i+1)
}

func treeNode(n *layout.Node, name func(int) string) *yaml.Node {
	m := &yaml.Node{Kind: yaml.MappingNode}
	if n.Leaf() {
		m.Style = yaml.FlowStyle
		m.Content = append(m.Content, str("pane"), str(name(n.Pane)))
	} else {
		key := "rows"
		if n.Dir == layout.Cols {
			key = "cols"
		}
		list := &yaml.Node{Kind: yaml.SequenceNode}
		for _, ch := range n.Children {
			list.Content = append(list.Content, treeNode(ch, name))
		}
		m.Content = append(m.Content, str(key), list)
	}
	if n.Size.Unit != layout.Auto {
		m.Content = append(m.Content, str("size"), str(sizeText(n.Size)))
	}
	return m
}

func paneNode(p, shared pane.Spec) *yaml.Node {
	m := &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle}
	kv := func(k, v string) { m.Content = append(m.Content, str(k), str(v)) }
	if p.Title != "" {
		kv("title", p.Title)
	}
	if p.Run != "" {
		kv("run", p.Run)
	}
	if p.Exec != "" {
		kv("exec", p.Exec)
	}
	if p.Restart != "" && p.Restart != pane.RestartNever {
		kv("restart", p.Restart)
	}
	if p.Shell != shared.Shell {
		kv("shell", p.Shell)
	}
	if p.Dir != shared.Dir {
		kv("cwd", p.Dir)
	}
	if strings.Join(p.Env, "\x00") != strings.Join(shared.Env, "\x00") {
		m.Content = append(m.Content, str("env"), envNode(p.Env))
	}
	return m
}

func envNode(env []string) *yaml.Node {
	sorted := append([]string(nil), env...)
	sort.Strings(sorted)
	m := &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle}
	for _, e := range sorted {
		k, v, _ := strings.Cut(e, "=")
		m.Content = append(m.Content, str(k), str(v))
	}
	return m
}

func sizeText(s layout.Size) string { return s.String() }

func str(v string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Value: v} }

func quoted(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: v, Style: yaml.DoubleQuotedStyle}
}

func flow(kv ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle, Content: kv}
}

func flowList(items []string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
	for _, it := range items {
		n.Content = append(n.Content, str(it))
	}
	return n
}
