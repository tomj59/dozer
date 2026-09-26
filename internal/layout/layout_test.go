package layout

import (
	"reflect"
	"testing"
)

func mustParse(t *testing.T, s string) *Node {
	t.Helper()
	n, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestParse(t *testing.T) {
	for spec, panes := range map[string]int{"default": 3, "2,1": 3, "2,2,1": 5, "1": 1, "3": 3, "": 3} {
		if got := mustParse(t, spec).Panes(); got != panes {
			t.Errorf("Parse(%q).Panes() = %d, want %d", spec, got, panes)
		}
	}
	for _, bad := range []string{"0", "a", "2,,1", "10", "3,3,3,1"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) should fail", bad)
		}
	}
}

func TestDefaultLayout(t *testing.T) {
	// 100x30 screen: top row 50% (15 rows), two panes split by a divider.
	r := Solve(mustParse(t, "2,1"), 100, 30, Min{20, 5})
	want := []Rect{{0, 0, 50, 15}, {51, 0, 49, 15}, {0, 15, 100, 15}}
	if !reflect.DeepEqual(r.Slots, want) {
		t.Errorf("slots = %v, want %v", r.Slots, want)
	}
	if !reflect.DeepEqual(r.Dividers, []Divider{{50, 0, 15}}) {
		t.Errorf("dividers = %v", r.Dividers)
	}
	if r.W != 100 || r.H != 30 {
		t.Errorf("canvas %dx%d, want screen size", r.W, r.H)
	}
	if c := r.Slots[2].Content(); c != (Rect{0, 16, 100, 14}) {
		t.Errorf("content = %v", c)
	}
}

func TestSizes(t *testing.T) {
	root := mustParse(t, "2,1")
	h, _ := ParseSizes("60,40")
	if err := SetHeights(root, h); err != nil {
		t.Fatal(err)
	}
	row, w, err := ParseWidths("1:30,70")
	if err != nil {
		t.Fatal(err)
	}
	if err := SetWidths(root, row, w); err != nil {
		t.Fatal(err)
	}
	r := Solve(root, 101, 40, Min{10, 3})
	// 101 wide minus 1 divider = 100 → 30 / 70; heights 24 / 16.
	want := []Rect{{0, 0, 30, 24}, {31, 0, 70, 24}, {0, 24, 101, 16}}
	if !reflect.DeepEqual(r.Slots, want) {
		t.Errorf("slots = %v, want %v", r.Slots, want)
	}
	if err := SetHeights(root, h[:1]); err == nil {
		t.Error("wrong height count should fail")
	}
	if err := SetWidths(root, 2, w); err == nil {
		t.Error("row 2 has one pane; two widths should fail")
	}
}

func TestDistribute(t *testing.T) {
	cases := []struct {
		total int
		sizes []Size
		mins  []int
		want  []int
	}{
		{10, []Size{{}, {}, {}}, []int{1, 1, 1}, []int{4, 3, 3}},
		{100, []Size{{20, Cells}, {}}, []int{1, 1}, []int{20, 80}},
		{100, []Size{{1, Weight}, {3, Weight}}, []int{1, 1}, []int{25, 75}},
		{100, []Size{{90, Percent}, {}}, []int{1, 30}, []int{70, 30}},           // min wins
		{100, []Size{{80, Percent}, {80, Percent}}, []int{1, 1}, []int{50, 50}}, // over-subscribed scales
	}
	for _, c := range cases {
		if got := Distribute(c.total, c.sizes, c.mins); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Distribute(%d, %v, %v) = %v, want %v", c.total, c.sizes, c.mins, got, c.want)
		}
	}
}

func TestTooSmallGrowsCanvas(t *testing.T) {
	// 3 columns of min 20 need 62 cells; the screen has 40.
	r := Solve(mustParse(t, "3"), 40, 10, Min{20, 5})
	if r.W != 62 || r.H != 10 {
		t.Fatalf("canvas %dx%d, want 62x10", r.W, r.H)
	}
	for i, s := range r.Slots {
		if s.W < 20 {
			t.Errorf("slot %d width %d below minimum", i, s.W)
		}
	}
}

func TestSizesCoverCanvas(t *testing.T) {
	// Every cell of the canvas belongs to exactly one slot or divider.
	for _, spec := range []string{"2,1", "2,2,1", "3,3,3", "1", "4"} {
		for _, dim := range [][2]int{{80, 24}, {173, 51}, {30, 12}} {
			r := Solve(mustParse(t, spec), dim[0], dim[1], Min{8, 2})
			grid := make([]int, r.W*r.H)
			for _, s := range r.Slots {
				for y := s.Y; y < s.Y+s.H; y++ {
					for x := s.X; x < s.X+s.W; x++ {
						grid[y*r.W+x]++
					}
				}
			}
			for _, d := range r.Dividers {
				for y := d.Y; y < d.Y+d.H; y++ {
					grid[y*r.W+d.X]++
				}
			}
			for i, c := range grid {
				if c != 1 {
					t.Fatalf("%s at %v: cell (%d,%d) covered %d times", spec, dim, i%r.W, i/r.W, c)
				}
			}
		}
	}
}

func TestMoveBorder(t *testing.T) {
	root := mustParse(t, "2,1")
	m := Min{W: 10, H: 3}
	solve := func() Result { return Solve(root, 101, 40, m) }

	// Pane 0 (top-left) → right: its right border moves right, it grows.
	if !MoveBorder(root, 0, 4, 0, solve(), m) {
		t.Fatal("no change")
	}
	if s := solve().Slots; s[0].W != 54 || s[1].W != 46 {
		t.Errorf("after →: widths %d/%d, want 54/46", s[0].W, s[1].W)
	}
	// Pane 1 (top-right, last child) → right: its left border moves right, it shrinks.
	MoveBorder(root, 1, 4, 0, solve(), m)
	if s := solve().Slots; s[0].W != 58 || s[1].W != 42 {
		t.Errorf("after pane 1 →: widths %d/%d, want 58/42", s[0].W, s[1].W)
	}
	// Pane 2 (bottom) ↑: its top border moves up, it grows; top row shrinks.
	MoveBorder(root, 2, 0, -5, solve(), m)
	if s := solve().Slots; s[0].H != 15 || s[2].H != 25 {
		t.Errorf("after ↑: heights %d/%d, want 15/25", s[0].H, s[2].H)
	}
	// Proportions survive a window resize (weights, not cells).
	if s := Solve(root, 201, 80, m).Slots; s[0].H != 30 {
		t.Errorf("after window resize: top height %d, want 30", s[0].H)
	}
	// Minimums stop the border.
	for i := 0; i < 50; i++ {
		MoveBorder(root, 0, -4, 0, solve(), m)
	}
	if s := solve().Slots; s[0].W != 10 {
		t.Errorf("min width not honored: %d", s[0].W)
	}
	if MoveBorder(root, 0, -4, 0, solve(), m) {
		t.Error("move past the minimum should report no change")
	}
	// A single-pane row has no vertical border to move.
	if MoveBorder(root, 2, 4, 0, solve(), m) {
		t.Error("pane 2 has no column split")
	}
}

func TestClone(t *testing.T) {
	a := mustParse(t, "2,1")
	b := a.Clone()
	b.Children[0].Size = Size{50, Cells}
	if a.Children[0].Size.Unit == Cells {
		t.Error("clone shares nodes")
	}
}
