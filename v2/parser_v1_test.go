package ogdl

import (
	"strings"
	"testing"
)

// TestParseV1Chain checks the v1 text mode against trees ogdl v1 produces
// for the same input (taken from v1.5.0's FromString).
func TestParseV1Chain(t *testing.T) {
	cases := []struct{ in, want string }{
		{"a b c\n", "a\n  b\n    c\n"},
		{"a b c\n  d\n", "a\n  b\n    c\n  d\n"},
		{"a b c\n    d\n", "a\n  b\n    c\n  d\n"},
		{"a b c\ne\n", "a\n  b\n    c\ne\n"},
		{"a b c d\n  e\n  f g h\n", "a\n  b\n    c\n      d\n  e\n  f\n    g\n      h\n"},
		{"x\n  a b c\n    d\n  e\n", "x\n  a\n    b\n      c\n    d\n  e\n"},
		{"a b\n  c d e\n", "a\n  b\n  c\n    d\n      e\n"},
		{"a 'b c' d\n", "a\n  \"b c\"\n    d\n"},
		{"a b \\\n  text\n", "a\n  b\n    text\n"},
		// Two elements: the same in both grammars.
		{"a b\n", "a\n  b\n"},
		{"S1  value 1.0  class asil\n", "S1\n  value\n    1.0\n      class\n        asil\n"},
	}
	for _, c := range cases {
		n, err := ParseV1(strings.NewReader(c.in))
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if got := n.Text(); got != c.want {
			t.Errorf("%q:\n got %q\nwant %q", c.in, got, c.want)
		}
		if got := FromString(c.in).Text(); got != c.want {
			t.Errorf("FromString %q:\n got %q\nwant %q", c.in, got, c.want)
		}
		if got := FromBytes([]byte(c.in)).Text(); got != c.want {
			t.Errorf("FromBytes %q:\n got %q\nwant %q", c.in, got, c.want)
		}
	}

	// The RFC grammar is unchanged.
	n, err := ParseString("a b c\n")
	if err != nil || n.Text() != "a\n  b\n  c\n" {
		t.Errorf("ParseString: %q %v", n.Text(), err)
	}
}

func TestParseV1Depth(t *testing.T) {
	p, _ := NewParser(strings.NewReader("a b c d e\n"))
	p.Chain = true
	p.MaxDepth = 3
	if _, err := p.Parse(); err == nil {
		t.Error("a chain deeper than MaxDepth must be rejected")
	}
}
