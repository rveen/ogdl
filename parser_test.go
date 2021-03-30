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
