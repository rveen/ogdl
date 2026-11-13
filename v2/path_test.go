package ogdl

import (
	"errors"
	"testing"
)

const pathDoc = `book
  chapter
    title First
    p one
    p two
  chapter
    title Second
    p three
list
  a
  b
  c
host
  '192.168.0.10' up
`

func pathTree(t *testing.T) *Node {
	t.Helper()
	g, err := ParseString(pathDoc)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestPathParse(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"a", "token(a)"},
		{"a.b", "token(a) token(b)"},
		{"a[1]", "token(a) index(1)"},
		{"a.[1]", "token(a) index(1)"},
		{"[0]", "index(0)"},
		{"a{}", "token(a) select(all)"},
		{"a{2}", "token(a) select(2)"},
		{"a{}.b{}", "token(a) select(all) token(b) select(all)"},
		{"'a.b'.c", "token(a.b) token(c)"},
		{`"a b".c`, "token(a b) token(c)"},
		{"a.{1}", "token(a) select(1)"},
		{"'a-b'.'c+d'", "token(a-b) token(c+d)"}, // operators require quoting
		{`'it\'s'`, "token(it's)"},               // escapes in quoted elements, 4.3
		{`"a\"b"`, `token(a"b)`},
		{"a[007]", "token(a) index(7)"}, // leading zeros are not significant
	}
	for _, c := range cases {
		got, err := Path(c.path)
		if err != nil {
			t.Errorf("Path(%q): %v", c.path, err)
			continue
		}
		if got != c.want {
			t.Errorf("Path(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestPathParseErrors(t *testing.T) {
	cases := []struct {
		path string
		want error
	}{
		{"", ErrEmptyPath},
		{"a.", ErrInvalidPath},
		{"a..b", ErrInvalidPath},
		{"a b", ErrInvalidPath},
		{"a[", ErrInvalidPath},
		{"a[]", ErrInvalidPath},
		{"a[x]", ErrInvalidPath},
		{"a{", ErrInvalidPath},
		{"{0}", ErrDanglingSelect},
		{"a[0]{1}", ErrDanglingSelect},
		{"a{}{0}", ErrDanglingSelect},
		{"'abc", ErrUnterminated},
		{`a.'b\qc'`, ErrInvalidEscape},
		// Everything outside letters, digits and _ must be quoted, so that the
		// characters stay free for the expression layer (Path RFC 4.2, 10).
		{"a-b", ErrInvalidPath},
		{"a+b", ErrInvalidPath},
		{"a(b)", ErrInvalidPath},
		{"a!b", ErrInvalidPath},
		{"a[٠]", ErrInvalidPath}, // non-ASCII digit
		{"a[99999999999]", ErrIndexTooLarge},
	}
	for _, c := range cases {
		_, err := Path(c.path)
		if err == nil {
			t.Errorf("Path(%q): expected %v, got none", c.path, c.want)
			continue
		}
		if !errors.Is(err, c.want) {
			t.Errorf("Path(%q): expected %v, got %v", c.path, c.want, err)
		}
		var pe *PathError
		if !errors.As(err, &pe) {
			t.Errorf("Path(%q): expected a *PathError, got %T", c.path, err)
		}
	}
}

func TestGet(t *testing.T) {
	g := pathTree(t)

	cases := []struct {
		path string
		this string // ThisString of the result
		str  string // String of the result
	}{
		{"book.chapter.title", "title", "First"},
		{"book.chapter{1}.title", "title", "Second"},
		{"book.chapter{0}.title", "title", "First"},
		{"list[0]", "a", ""},
		{"list[2]", "c", ""},
		{"book.chapter[0]", "title", "First"},
		{"book.chapter[1]", "p", "one"},
		{"[0]", "book", "chapter"},
		{"book", "book", "chapter"},
		{"host.'192.168.0.10'", "192.168.0.10", "up"},
		{"missing.x", "", ""},
		{"list[9]", "", ""},
		{"book.chapter{9}.title", "", ""},
	}

	for _, c := range cases {
		r := g.Get(c.path)
		if r == nil {
			t.Errorf("Get(%q) returned nil; it must never be nil", c.path)
			continue
		}
		if got := r.ThisString(); got != c.this {
			t.Errorf("Get(%q).ThisString() = %q, want %q", c.path, got, c.this)
		}
		if got := r.String(); got != c.str {
			t.Errorf("Get(%q).String() = %q, want %q", c.path, got, c.str)
		}
	}
}

func TestGetSet(t *testing.T) {
	g := pathTree(t)

	// The empty selector yields a set: a node whose children are the matches.
	texts := func(path string) []string {
		var out []string
		for _, m := range g.Get(path).Out {
			out = append(out, m.ThisString()+"="+m.String())
		}
		return out
	}

	got := texts("book.chapter.p{}")
	want := []string{"p=one", "p=two"}
	if !equalStrings(got, want) {
		t.Errorf("book.chapter.p{} = %v, want %v", got, want)
	}

	got = texts("book.chapter{}")
	want = []string{"chapter=title", "chapter=title"}
	if !equalStrings(got, want) {
		t.Errorf("book.chapter{} = %v, want %v", got, want)
	}

	// A token after a set maps over its members, one match each.
	got = texts("book.chapter{}.title")
	want = []string{"title=First", "title=Second"}
	if !equalStrings(got, want) {
		t.Errorf("book.chapter{}.title = %v, want %v", got, want)
	}

	got = texts("book.chapter{}.title{}")
	want = []string{"title=First", "title=Second"}
	if !equalStrings(got, want) {
		t.Errorf("book.chapter{}.title{} = %v, want %v", got, want)
	}

	// Every p of every chapter.
	got = texts("book.chapter{}.p{}")
	want = []string{"p=one", "p=two", "p=three"}
	if !equalStrings(got, want) {
		t.Errorf("book.chapter{}.p{} = %v, want %v", got, want)
	}
}

func TestGetErr(t *testing.T) {
	g := pathTree(t)

	if _, err := g.GetErr("a b"); err == nil {
		t.Error("expected a parse error")
	}
	// Not found is not an error.
	r, err := g.GetErr("nothing.here")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if r == nil || len(r.Out) != 0 || r.ThisString() != "" {
		t.Errorf("expected an empty node, got %#v", r)
	}
}

func TestGetOnNonStringData(t *testing.T) {
	root := &Node{Out: []*Node{
		{Data: "port", Out: []*Node{{Data: 8080}}},
	}}
	if got := root.Get("port").String(); got != "8080" {
		t.Errorf(`Get("port").String() = %q, want "8080"`, got)
	}
	// A node holding an int is addressable by its digits.
	if got := root.Get("port.8080").ThisString(); got != "8080" {
		t.Errorf(`Get("port.8080").ThisString() = %q, want "8080"`, got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
