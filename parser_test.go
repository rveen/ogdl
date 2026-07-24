package ogdl

import (
	"fmt"
	"testing"
)

func TestParser(t *testing.T) {

	cases := []struct {
		in   string
		want string
	}{
		{"a", "a"},
		{"\na", "a"},
		{"a\n", "a"},
		{"a    ", "a"},
		{" a", "a"},
		{"a    \n", "a"},
		{"a\nb", "a\nb"},
		{"a b", "a\n  b"},
		{"a\r\nb", "a\nb"},
		{"a b c d", "a\n  b\n    c\n      d"},

		// Comments

		{"# comment", ""},
		{"# comment\nnot#acomment", "not#acomment"},

		// Blocks

		{"a \\\n  b\n  c", "a\n  \"b\n  c\""},
		{"a \\\n  b c", "a\n  \"b c\""},

		// A line that ends a block keeps its indentation: 'c' is a child
		// of 'a', not a new root level node.
		{"a\n  b \\\n    text\n  c", "a\n  b\n    text\n  c"},
		{"a\n  b\n    c \\\n      text\n  d", "a\n  b\n    c\n      text\n  d"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s in %s", tc.in, tc.want), func(t *testing.T) {
			g := FromString(tc.in)
			got := g.Text()
			if got != tc.want {
				t.Errorf("got %s; want %s", got, tc.want)
			}
		})
	}

}

// TestBlockIndent checks the content of a block: the indentation of its first
// line is stripped from all lines, and indentation relative to that first line
// is preserved.
func TestBlockIndent(t *testing.T) {

	cases := []struct {
		in   string
		path string
		want string
	}{
		{"a \\\n  b\n  c", "a", "b\nc"},
		{"a \\\n  b\n    c\n  d", "a", "b\n  c\nd"},

		// The block does not depend on the level at which it appears.
		{"x\n  y\n    a \\\n      b\n        c\n      d", "x.y.a", "b\n  c\nd"},

		// Blank lines are kept, and do not set the base indentation.
		{"a \\\n  b\n\n    c", "a", "b\n\n  c"},

		// Tabs.
		{"a \\\n\tb\n\t\tc", "a", "b\n\tc"},

		// A line indented less than the first one gets no indentation.
		{"a \\\n    b\n  c\n    d", "a", "b\nc\nd"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q", tc.in), func(t *testing.T) {
			got := FromString(tc.in).Get(tc.path).String()
			if got != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}
