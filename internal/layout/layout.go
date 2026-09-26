// Package layout turns a layout description (the "2,2,1" shorthand plus
// optional sizes) into a tree, and solves that tree into screen rectangles.
//
// Geometry rules (docs/SPEC.md §4.2, §4.6):
//   - Every pane slot starts with a one-row title divider; the content area
//     is the rest of the slot.
//   - Side-by-side slots are separated by a one-column vertical divider.
//   - Stacked slots need no extra divider: the lower slot's title row is
//     the separator.
//   - Slots never shrink below a minimum content size. If the screen is too
//     small, the solver returns a canvas larger than the screen, and the UI
//     shows a scrolling viewport onto it (DP-1).
package layout

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Dir is the direction a split lays out its children.
type Dir int

const (
	// Rows stacks children top to bottom.
	Rows Dir = iota
	// Cols places children left to right.
	Cols
)

// Unit is the unit of a Size.
type Unit int

const (
	// Auto shares leftover space equally (weight 1).
	Auto Unit = iota
	// Percent of the parent's available space.
	Percent
	// Cells is a fixed number of rows or columns.
	Cells
	// Weight shares leftover space in proportion to Value.
	Weight
)

// Size is a node's size along its parent's split direction.
type Size struct {
	Value float64
	Unit  Unit
}

func (s Size) String() string {
	switch s.Unit {
	case Percent:
		return strconv.FormatFloat(s.Value, 'f', -1, 64) + "%"
	case Cells:
		return strconv.FormatFloat(s.Value, 'f', -1, 64) + "c"
	case Weight:
		return strconv.FormatFloat(s.Value, 'f', -1, 64) + "fr"
	}
	return "auto"
}

// ParseSize parses "60", "60%", "20c" or "2fr". A bare number is a percent.
func ParseSize(s string) (Size, error) {
	s = strings.TrimSpace(s)
	if s == "auto" {
		return Size{}, nil
	}
	unit := Percent
	switch {
	case strings.HasSuffix(s, "%"):
		s = strings.TrimSuffix(s, "%")
	case strings.HasSuffix(s, "fr"):
		s, unit = strings.TrimSuffix(s, "fr"), Weight
	case strings.HasSuffix(s, "c"):
		s, unit = strings.TrimSuffix(s, "c"), Cells
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 || math.IsInf(v, 0) {
		return Size{}, fmt.Errorf("bad size %q (use 60, 60%%, 20c, 2fr or auto)", s)
	}
	return Size{Value: v, Unit: unit}, nil
}

// ParseSizes parses a comma-separated size list such as "60,40".
func ParseSizes(s string) ([]Size, error) {
	var out []Size
	for _, f := range strings.Split(s, ",") {
		sz, err := ParseSize(f)
		if err != nil {
			return nil, err
		}
		out = append(out, sz)
	}
	return out, nil
}

// Node is a layout tree node: either a pane slot (leaf) or a split.
type Node struct {
	Pane     int // leaf: pane index in reading order; split: -1
	Dir      Dir
	Children []*Node
	Size     Size
}

// Leaf reports whether n is a pane slot.
func (n *Node) Leaf() bool { return n.Pane >= 0 }

// Panes returns the number of pane slots in the tree.
func (n *Node) Panes() int {
	if n.Leaf() {
		return 1
	}
	c := 0
	for _, ch := range n.Children {
		c += ch.Panes()
	}
	return c
}

// Presets are named layouts. "default" is two panes on top, one wide below.
var Presets = map[string]string{
	"default": "2,1",
}

// Parse builds a tree from row shorthand: comma-separated pane counts per
// row, for example "2,1" or "2,2,1". A preset name is also accepted.
func Parse(spec string) (*Node, error) {
	if p, ok := Presets[spec]; ok {
		spec = p
	}
	if strings.TrimSpace(spec) == "" {
		spec = Presets["default"]
	}
	root := &Node{Pane: -1, Dir: Rows}
	idx := 0
	for _, f := range strings.Split(spec, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil || n < 1 || n > 9 {
			return nil, fmt.Errorf("bad layout %q: each row needs 1-9 panes, e.g. 2,1 or 2,2,1", spec)
		}
		if n == 1 {
			root.Children = append(root.Children, &Node{Pane: idx})
			idx++
			continue
		}
		row := &Node{Pane: -1, Dir: Cols}
		for i := 0; i < n; i++ {
			row.Children = append(row.Children, &Node{Pane: idx})
			idx++
		}
		root.Children = append(root.Children, row)
	}
	if idx > 9 {
		return nil, fmt.Errorf("layout %q has %d panes; the maximum is 9", spec, idx)
	}
	if len(root.Children) == 1 { // a single row: no need for the outer split
		only := root.Children[0]
		only.Size = Size{}
		return only, nil
	}
	return root, nil
}

// SetHeights applies row sizes to a shorthand tree ("--heights 60,40").
func SetHeights(root *Node, sizes []Size) error {
	if root.Leaf() || root.Dir != Rows {
		if len(sizes) == 1 {
			return nil
		}
		return fmt.Errorf("--heights: layout has 1 row, got %d sizes", len(sizes))
	}
	if len(sizes) != len(root.Children) {
		return fmt.Errorf("--heights: layout has %d rows, got %d sizes", len(root.Children), len(sizes))
	}
	for i, s := range sizes {
		root.Children[i].Size = s
	}
	return nil
}

// SetWidths applies column sizes to row r (1-based) of a shorthand tree
// ("--widths 1:30,70").
func SetWidths(root *Node, r int, sizes []Size) error {
	row := root
	if !root.Leaf() && root.Dir == Rows {
		if r < 1 || r > len(root.Children) {
			return fmt.Errorf("--widths: no row %d (layout has %d rows)", r, len(root.Children))
		}
		row = root.Children[r-1]
	} else if r != 1 {
		return fmt.Errorf("--widths: no row %d (layout has 1 row)", r)
	}
	if row.Leaf() {
		if len(sizes) == 1 {
			return nil
		}
		return fmt.Errorf("--widths: row %d has 1 pane, got %d sizes", r, len(sizes))
	}
	if len(sizes) != len(row.Children) {
		return fmt.Errorf("--widths: row %d has %d panes, got %d sizes", r, len(row.Children), len(sizes))
	}
	for i, s := range sizes {
		row.Children[i].Size = s
	}
	return nil
}

// ParseWidths parses "R:sizes", for example "1:30,70".
func ParseWidths(s string) (row int, sizes []Size, err error) {
	r, list, ok := strings.Cut(s, ":")
	if !ok {
		return 0, nil, fmt.Errorf("--widths %q: use ROW:SIZES, e.g. 1:30,70", s)
	}
	row, err = strconv.Atoi(strings.TrimSpace(r))
	if err != nil {
		return 0, nil, fmt.Errorf("--widths %q: bad row number", s)
	}
	sizes, err = ParseSizes(list)
	return row, sizes, err
}

// Rect is a rectangle in cells.
type Rect struct{ X, Y, W, H int }

// Content returns the slot's content area (below its title row).
func (r Rect) Content() Rect { return Rect{r.X, r.Y + 1, r.W, max(r.H-1, 0)} }

// Divider is a vertical divider column between side-by-side slots.
type Divider struct{ X, Y, H int }

// Result is a solved layout.
type Result struct {
	Slots    []Rect    // indexed by pane number; each includes its title row
	Dividers []Divider // vertical dividers
	W, H     int       // canvas size; larger than the screen if it didn't fit
}

// Min is the minimum content size of a pane.
type Min struct{ W, H int }

// minSize returns the minimum width and height of n's area.
func minSize(n *Node, m Min) (int, int) {
	if n.Leaf() {
		return m.W, m.H + 1 // + title row
	}
	w, h := 0, 0
	for i, ch := range n.Children {
		cw, chh := minSize(ch, m)
		if n.Dir == Cols {
			w += cw
			if i > 0 {
				w++ // divider
			}
			h = max(h, chh)
		} else {
			h += chh
			w = max(w, cw)
		}
	}
	return w, h
}

// Solve lays root out in a w×h area with minimum pane size m.
func Solve(root *Node, w, h int, m Min) Result {
	mw, mh := minSize(root, m)
	res := Result{Slots: make([]Rect, root.Panes()), W: max(w, mw), H: max(h, mh)}
	solve(root, Rect{0, 0, res.W, res.H}, m, &res)
	return res
}

func solve(n *Node, r Rect, m Min, res *Result) {
	if n.Leaf() {
		if n.Pane < len(res.Slots) {
			res.Slots[n.Pane] = r
		}
		return
	}
	k := len(n.Children)
	total := r.H
	if n.Dir == Cols {
		total = r.W - (k - 1) // dividers
	}
	sizes := make([]Size, k)
	mins := make([]int, k)
	for i, ch := range n.Children {
		sizes[i] = ch.Size
		cw, chh := minSize(ch, m)
		mins[i] = chh
		if n.Dir == Cols {
			mins[i] = cw
		}
	}
	lens := Distribute(total, sizes, mins)
	pos := r.X
	if n.Dir == Rows {
		pos = r.Y
	}
	for i, ch := range n.Children {
		if n.Dir == Cols {
			if i > 0 {
				res.Dividers = append(res.Dividers, Divider{X: pos, Y: r.Y, H: r.H})
				pos++
			}
			solve(ch, Rect{pos, r.Y, lens[i], r.H}, m, res)
		} else {
			solve(ch, Rect{r.X, pos, r.W, lens[i]}, m, res)
		}
		pos += lens[i]
	}
}

// Distribute splits total cells among children with the given sizes,
// honoring minimums. Fixed cells and percents are taken first, weights and
// Auto share the rest; the result is then scaled to fill total exactly and
// rounded by largest remainder. The sum of mins must not exceed total.
func Distribute(total int, sizes []Size, mins []int) []int {
	k := len(sizes)
	want := make([]float64, k)
	used, weights := 0.0, 0.0
	for i, s := range sizes {
		switch s.Unit {
		case Cells:
			want[i] = s.Value
			used += s.Value
		case Percent:
			want[i] = s.Value / 100 * float64(total)
			used += want[i]
		case Weight:
			weights += s.Value
		default:
			weights++
		}
	}
	if weights > 0 {
		rest := math.Max(float64(total)-used, 0)
		for i, s := range sizes {
			switch s.Unit {
			case Weight:
				want[i] = rest * s.Value / weights
			case Auto:
				want[i] = rest / weights
			}
		}
	}

	out := make([]int, k)
	fixed := make([]bool, k) // pinned at their minimum
	for {
		// Scale the unpinned wants to fill what the pinned ones leave.
		avail, sum := float64(total), 0.0
		for i := range want {
			if fixed[i] {
				avail -= float64(out[i])
			} else {
				sum += want[i]
			}
		}
		scaled := make([]float64, k)
		for i := range want {
			if fixed[i] {
				continue
			}
			if sum > 0 {
				scaled[i] = want[i] / sum * avail
			} else {
				scaled[i] = avail / float64(countFalse(fixed))
			}
		}
		roundInto(out, scaled, fixed, int(math.Round(avail)))
		violated := false
		for i := range out {
			if !fixed[i] && out[i] < mins[i] {
				out[i], fixed[i], violated = mins[i], true, true
			}
		}
		if !violated {
			return out
		}
	}
}

func countFalse(b []bool) int {
	n := 0
	for _, v := range b {
		if !v {
			n++
		}
	}
	return max(n, 1)
}

// roundInto rounds the unfixed entries of f so they sum to target, using
// the largest-remainder method.
func roundInto(out []int, f []float64, fixed []bool, target int) {
	sum := 0
	type rem struct {
		i int
		r float64
	}
	var rems []rem
	for i, v := range f {
		if fixed[i] {
			continue
		}
		out[i] = int(math.Floor(v))
		sum += out[i]
		rems = append(rems, rem{i, v - math.Floor(v)})
	}
	for left := target - sum; left > 0 && len(rems) > 0; left-- {
		best := 0
		for j := range rems {
			if rems[j].r > rems[best].r {
				best = j
			}
		}
		out[rems[best].i]++
		rems[best].r = -1
	}
}

// Clone returns a deep copy of the tree.
func (n *Node) Clone() *Node {
	c := *n
	c.Children = nil
	for _, ch := range n.Children {
		c.Children = append(c.Children, ch.Clone())
	}
	return &c
}

// Extent is the rectangle covered by n's pane slots in a solved layout.
func Extent(n *Node, r Result) Rect {
	if n.Leaf() {
		return r.Slots[n.Pane]
	}
	e := Extent(n.Children[0], r)
	for _, ch := range n.Children[1:] {
		o := Extent(ch, r)
		x0, y0 := min(e.X, o.X), min(e.Y, o.Y)
		x1, y1 := max(e.X+e.W, o.X+o.W), max(e.Y+e.H, o.Y+o.H)
		e = Rect{x0, y0, x1 - x0, y1 - y0}
	}
	return e
}

// path returns the nodes from root down to pane's leaf.
func path(n *Node, pane int) []*Node {
	if n.Leaf() {
		if n.Pane == pane {
			return []*Node{n}
		}
		return nil
	}
	for _, ch := range n.Children {
		if p := path(ch, pane); p != nil {
			return append([]*Node{n}, p...)
		}
	}
	return nil
}

// MoveBorder moves the border of pane's area in direction (dx, dy) by
// |dx| columns or |dy| rows, the way tmux's resize-pane does. The border
// on the side the move points to is used, or the opposite border if the
// pane touches that edge of its split. It works on the nearest enclosing
// split of the right direction. Sizes of that split become weights equal
// to their current cells, so the new proportions survive window resizes.
// It reports whether anything changed.
func MoveBorder(root *Node, pane, dx, dy int, r Result, m Min) bool {
	dir, delta := Cols, dx
	if dy != 0 {
		dir, delta = Rows, dy
	}
	if delta == 0 {
		return false
	}
	p := path(root, pane)
	for i := len(p) - 2; i >= 0; i-- {
		split := p[i]
		if split.Dir != dir || len(split.Children) < 2 {
			continue
		}
		k := indexOf(split.Children, p[i+1])
		// Border between children a and a+1.
		// Use the border on the side the move points to; if the pane is at
		// that edge of the split, use the border on its other side.
		a := k // border after child k
		if delta < 0 {
			a = k - 1 // border before child k
		}
		if a < 0 {
			a = 0
		}
		if a >= len(split.Children)-1 {
			a = len(split.Children) - 2
		}
		if a < 0 || a+1 >= len(split.Children) {
			continue
		}
		lens := make([]int, len(split.Children))
		mins := make([]int, len(split.Children))
		for j, ch := range split.Children {
			e := Extent(ch, r)
			mw, mh := minSize(ch, m)
			if dir == Cols {
				lens[j], mins[j] = e.W, mw
			} else {
				lens[j], mins[j] = e.H, mh
			}
		}
		// Moving the border by delta grows child a and shrinks a+1.
		d := delta
		if d > 0 {
			d = min(d, lens[a+1]-mins[a+1])
		} else {
			d = max(d, mins[a]-lens[a])
		}
		if d == 0 {
			return false
		}
		lens[a] += d
		lens[a+1] -= d
		for j, ch := range split.Children {
			ch.Size = Size{Value: float64(lens[j]), Unit: Weight}
		}
		return true
	}
	return false
}

func indexOf(list []*Node, n *Node) int {
	for i, x := range list {
		if x == n {
			return i
		}
	}
	return -1
}
