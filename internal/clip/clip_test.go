package clip

import (
	"errors"
	"reflect"
	"testing"
)

func TestCommand(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	have := func(names ...string) func(string) (string, error) {
		return func(n string) (string, error) {
			for _, x := range names {
				if x == n {
					return "/usr/bin/" + n, nil
				}
			}
			return "", errors.New("not found")
		}
	}
	cases := []struct {
		goos string
		env  map[string]string
		bins []string
		want []string
	}{
		{"darwin", nil, []string{"pbcopy"}, []string{"pbcopy"}},
		{"darwin", map[string]string{"SSH_CONNECTION": "1 2 3 4"}, []string{"pbcopy"}, nil},
		{"linux", map[string]string{"WAYLAND_DISPLAY": "w"}, []string{"wl-copy", "xclip"}, []string{"wl-copy"}},
		{"linux", map[string]string{"DISPLAY": ":0"}, []string{"xclip"}, []string{"xclip", "-selection", "clipboard"}},
		{"linux", map[string]string{"DISPLAY": ":0"}, []string{"xsel"}, []string{"xsel", "--clipboard", "--input"}},
		{"linux", nil, []string{"xclip"}, nil}, // no display: a console or a server
	}
	for _, c := range cases {
		if got := Command(c.goos, env(c.env), have(c.bins...)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s %v %v: got %v want %v", c.goos, c.env, c.bins, got, c.want)
		}
	}
}

func TestLocal(t *testing.T) {
	if err := Local([]string{"sh", "-c", `test "$(cat)" = hello`}, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := Local([]string{"sh", "-c", "echo nope >&2; exit 1"}, "x"); err == nil || err.Error() != "sh: nope" {
		t.Fatalf("err = %v", err)
	}
}
