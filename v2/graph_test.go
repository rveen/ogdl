package ogdl

import (
	"testing"
)

func TestNameString(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{"abc", "abc"},
		{[]byte("abc"), "abc"},
		{42, "42"},
		{-1, "-1"},
		{3.14, "3.14"},
		{true, "true"},
		{nil, ""},
		{&Node{Data: "parent", Out: []*Node{{Data: "child"}}}, "child"},
	}
	for _, c := range cases {
		if got := nameString(c.in); got != c.want {
			t.Errorf("nameString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestNonStringData checks that data of any type serializes through its textual
// form, and is quoted by the same rules as a parsed name.
func TestNonStringData(t *testing.T) {
	root := &Node{Out: []*Node{
		{Data: "port", Out: []*Node{{Data: 8080}}},
		{Data: "ratio", Out: []*Node{{Data: 0.5}}},
		{Data: "debug", Out: []*Node{{Data: true}}},
		{Data: "raw", Out: []*Node{{Data: []byte("two words")}}},
		{Data: nil},
	}}

	got, err := root.Canonical()
	if err != nil {
		t.Fatal(err)
	}
	want := "port\n  8080\nratio\n  0.5\ndebug\n  true\nraw\n  \"two words\"\n\"\"\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}

	// Reparsing yields the same text, but every node now holds a string: a
	// stream carries no types (RFC 2.4).
	reparsed, err := ParseString(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := reparsed.Canonical()
	if err != nil {
		t.Fatal(err)
	}
	if again != want {
		t.Fatalf("round trip changed the text:\n%q\n%q", again, want)
	}
	if _, ok := reparsed.N(0).N(0).Data.(string); !ok {
		t.Fatalf("parsed data should be a string, got %T", reparsed.N(0).N(0).Data)
	}
}

// TestLookupByText checks that S matches on the textual form, so a node holding
// an int is found by its digits.
func TestLookupByText(t *testing.T) {
	root := &Node{Out: []*Node{
		{Data: 8080, Out: []*Node{{Data: "http"}}},
		{Data: "name", Out: []*Node{{Data: "value"}}},
	}}

	if got := root.S("8080").String(); got != "http" {
		t.Errorf(`S("8080").String() = %q, want "http"`, got)
	}
	if got := root.S("name").String(); got != "value" {
		t.Errorf(`S("name").String() = %q, want "value"`, got)
	}
	if got := root.S("missing").ThisString(); got != "" {
		t.Errorf("missing child should give an empty node, got %q", got)
	}
}

func TestThisStringNil(t *testing.T) {
	var n *Node
	if got := n.ThisString(); got != "" {
		t.Errorf("nil node ThisString = %q, want empty", got)
	}
	if got := n.String(); got != "" {
		t.Errorf("nil node String = %q, want empty", got)
	}
}
