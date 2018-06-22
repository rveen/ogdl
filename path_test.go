package ogdl

import (
	"fmt"
	"testing"
)

func TestPathRepresentation(t *testing.T) {

	p := NewPath("a")

	if p.Len() != 1 {
		t.Error("size != 1")
	}

	cases := []struct {
		name string
		path string
		want string
	}{
		{"simple", "a.b", "!p\n  a\n  b"},
		{"only one index", "[0]", "!p\n  !i\n    0"},
		{"index", "a[1]", "!p\n  a\n  !i\n    1"},
		{"func", "a(b)", "!p\n  a\n  !a\n    !e\n      !p\n        b"},
		{"eval arg", "a.(b)", "!p\n  a\n  !g\n    !e\n      !p\n        b"},
		{"flow syntax", "(a, b)", "!p\n  !g\n    !e\n      !p\n        a\n    !e\n      !p\n        b"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s in %s", tc.path, tc.want), func(t *testing.T) {
			g := NewPath(tc.path)
			got := g.Show()
			if got != tc.want {
				t.Errorf("got %s; want %s", got, tc.want)
			}
		})
	}
}
