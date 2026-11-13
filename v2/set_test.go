package ogdl

import (
	"errors"
	"testing"
)

func TestSet(t *testing.T) {
	cases := []struct {
		name  string
		doc   string
		path  string
		value interface{}
		want  string
	}{
		{
			name:  "replace content",
			doc:   "a\n  b\n    c\n",
			path:  "a.b",
			value: "d",
			want:  "a\n  b\n    d\n",
		},
		{
			name:  "create missing elements",
			doc:   "",
			path:  "x.y.z",
			value: 1,
			want:  "x\n  y\n    z\n      1\n",
		},
		{
			name:  "append at one past the end",
			doc:   "a\n  b\n    c\n",
			path:  "a[1]",
			value: "d",
			want:  "a\n  b\n    c\n  d\n",
		},
		{
			name:  "replace by index",
			doc:   "list\n  10\n  20\n  30\n",
			path:  "list[1]",
			value: 99,
			want:  "list\n  10\n  99\n  30\n",
		},
		{
			name:  "select an occurrence",
			doc:   "t\n  x 1\n  x 2\n",
			path:  "t.x{1}",
			value: 9,
			want:  "t\n  x\n    1\n  x\n    9\n",
		},
		{
			name:  "existing children are replaced",
			doc:   "a\n  b\n    one\n    two\n",
			path:  "a.b",
			value: "only",
			want:  "a\n  b\n    only\n",
		},
		{
			name:  "quoted element",
			doc:   "host\n",
			path:  "host.'192.168.0.10'",
			value: "up",
			want:  "host\n  192.168.0.10\n    up\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g, err := ParseString(c.doc)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := g.SetErr(c.path, c.value); err != nil {
				t.Fatalf("SetErr: %v", err)
			}
			got, err := g.Canonical()
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, c.want)
			}
		})
	}
}

func TestSetErrors(t *testing.T) {
	cases := []struct {
		doc  string
		path string
		want error
	}{
		{"list\n  10\n  20\n", "list[9]", ErrTargetIndex},
		{"t\n  x 1\n", "t.x{5}", ErrTargetMissing},
		{"t\n  x 1\n", "t.x{}", ErrTargetIsSet},
		{"a b\n", "a b", ErrInvalidPath},
		{"a b\n", "", ErrEmptyPath},
	}

	for _, c := range cases {
		g, err := ParseString(c.doc)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := g.Canonical()

		if _, err := g.SetErr(c.path, 1); !errors.Is(err, c.want) {
			t.Errorf("Set(%q): expected %v, got %v", c.path, c.want, err)
		}

		// A rejected assignment leaves the tree exactly as it was
		// (Expr RFC 6.6).
		after, _ := g.Canonical()
		if before != after {
			t.Errorf("Set(%q) modified the tree despite failing:\n%q\n%q", c.path, before, after)
		}
	}
}

// A failed target must not leave the nodes it created on the way (Expr RFC 6.6).
func TestSetRollsBackCreation(t *testing.T) {
	g, err := ParseString("t\n  x 1\n")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := g.Canonical()

	if _, err := g.SetErr("t.fresh{5}", 1); err == nil {
		t.Fatal("expected an error")
	}

	after, _ := g.Canonical()
	if before != after {
		t.Fatalf("creation was not rolled back:\n%q\n%q", before, after)
	}
}

func TestSetNodeValue(t *testing.T) {
	g, err := ParseString("a b\n")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := ParseString("one\ntwo\n")
	if err != nil {
		t.Fatal(err)
	}

	// A node value contributes its children, not itself.
	g.Set("a", sub)

	got, _ := g.Canonical()
	want := "a\n  one\n  two\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSetChaining(t *testing.T) {
	g := &Node{}
	// Set returns the node written to, and never nil.
	if n := g.Set("a.b", 1); n.ThisString() != "b" {
		t.Errorf("Set returned %q, want the target node", n.ThisString())
	}
	if n := g.Set("a b", 1); n == nil || len(n.Out) != 0 {
		t.Error("a malformed path should give an empty node, not nil")
	}
	if got := g.Get("a.b").String(); got != "1" {
		t.Errorf("a.b = %q, want 1", got)
	}
}
