package ogdl

import (
	"errors"
	"strings"
	"testing"
)

// formatDoc parses s as a document and formats it back.
func formatDoc(t *testing.T, s string) (*Document, string) {
	t.Helper()
	d, err := ParseDocumentString(s)
	if err != nil {
		t.Fatalf("parse error for %q: %v", s, err)
	}
	out, err := d.Format()
	if err != nil {
		t.Fatalf("format error for %q: %v", s, err)
	}
	return d, out
}

const config = `# server settings
server
  host db.example.com # primary
  port 8080

  # first match wins
  rule allow 10.0.0.0/8
  rule allow 192.168.0.0/16
  rule deny any
  # add more rules above
limits
  S1
    value 1.0
    class asil
  labels
    low
    high

# end of file
`

// TestDocumentRoundTrip checks that a document already in normalized layout
// comes back byte for byte: comments, blank lines and inline layout included.
func TestDocumentRoundTrip(t *testing.T) {
	_, out := formatDoc(t, config)
	if out != config {
		t.Fatalf("got:\n%s\nwant:\n%s", out, config)
	}
}

// TestDocumentSameTree checks that retaining presentation does not change the
// tree.
func TestDocumentSameTree(t *testing.T) {
	d, out := formatDoc(t, config)
	plain, err := ParseString(config)
	if err != nil {
		t.Fatal(err)
	}
	if d.Root.Text() != plain.Text() {
		t.Fatalf("document tree differs:\n%s\nvs\n%s", d.Root.Text(), plain.Text())
	}
	again, err := ParseString(out)
	if err != nil {
		t.Fatal(err)
	}
	if again.Text() != plain.Text() {
		t.Fatalf("formatted output denotes a different tree")
	}
}

// TestDocumentEdit changes the tree and checks that comments survive.
func TestDocumentEdit(t *testing.T) {
	d, err := ParseDocumentString(config)
	if err != nil {
		t.Fatal(err)
	}
	d.Root.Set("server.port", 9090)
	server := d.Root.S("server")
	server.Out = append(server.Out, &Node{Data: "rule", Out: []*Node{{Data: "deny"}, {Data: "all"}}})

	out, err := d.Format()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(config, "port 8080", "port 9090", 1)
	want = strings.Replace(want, "  rule deny any\n", "  rule deny any\n  rule deny all\n", 1)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

// TestDocumentKeepsLayout checks that a node written on its own line stays
// there, and an inline one stays inline, whatever the compact rule would do.
func TestDocumentKeepsLayout(t *testing.T) {
	in := "port\n  8080\nrule allow any\n"
	if _, out := formatDoc(t, in); out != in {
		t.Fatalf("got %q, want %q", out, in)
	}
}

// TestDocumentNormalizes covers what Format does not preserve: indentation,
// spacing, quote style, blocks and repeated blank lines.
func TestDocumentNormalizes(t *testing.T) {
	in := "\n\na\n\tb   'x y'   #  note\n\n\n\tc \\\n\t\tline one\n\t\tline two\n#\n"
	want := "a\n  b \"x y\" #  note\n\n  c \"line one\\nline two\"\n#\n"
	if _, out := formatDoc(t, in); out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

// TestBlankLineAfterBlock checks that blank lines ending a block are kept as
// presentation, not swallowed by the block.
func TestBlankLineAfterBlock(t *testing.T) {
	in := "text \\\n  hello\n\n# next\nnext\n"
	want := "text hello\n\n# next\nnext\n"
	if _, out := formatDoc(t, in); out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
	if got := canonical(t, in); got != "text\n  hello\nnext\n" {
		t.Fatalf("canonical: %q", got)
	}
}

func TestCommentAttachment(t *testing.T) {
	in := `a
  b
    c
    # end of b
  # end of a
# before d
d
  # end of d, at EOF
`
	d, out := formatDoc(t, in)
	if out != in {
		t.Fatalf("got:\n%s\nwant:\n%s", out, in)
	}
	a := d.Root.S("a")
	check := func(got []string, want ...string) {
		t.Helper()
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("got %q, want %q", got, want)
		}
	}
	check(d.Comments[a.S("b")].After, "# end of b")
	check(d.Comments[a].After, "# end of a")
	check(d.Comments[d.Root.S("d")].Before, "# before d")
	check(d.Comments[d.Root.S("d")].After, "# end of d, at EOF")
	if c := d.Comments[d.Root]; c != nil {
		t.Errorf("root should have no comments, got %+v", c)
	}
}

// TestCommentOrderPreserved checks that a comment indented deeper than one
// before it is not moved above it.
func TestCommentOrderPreserved(t *testing.T) {
	in := "a\n  b\n    c\n  # one\n    # two\nd\n"
	want := "a\n  b\n    c\n  # one\n  # two\nd\n"
	if _, out := formatDoc(t, in); out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestProgrammaticComments(t *testing.T) {
	root := &Node{Out: []*Node{{Data: "a", Out: []*Node{{Data: "1"}}}}}
	d := &Document{Root: root, Comments: map[*Node]*Comments{
		root.Out[0]: {Before: []string{"plain text", "#tight", "two\nlines"}, Line: "trailing"},
		root:        {After: []string{"", "# bye"}},
	}}
	want := "# plain text\n# tight\n# two\n# lines\na 1 # trailing\n\n# bye\n"
	out, err := d.Format()
	if err != nil {
		t.Fatal(err)
	}
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}

	d.Comments[root.Out[0]].Line = "bad\x00"
	if _, err := d.Format(); !errors.Is(err, ErrControlCharacter) {
		t.Fatalf("expected ErrControlCharacter, got %v", err)
	}
}

// A commented node is never written inline, since its comments need a line.
func TestCommentedNodeNotInline(t *testing.T) {
	root := &Node{Out: []*Node{{Data: "a", Out: []*Node{{Data: "b"}, {Data: "c"}}}}}
	d := &Document{Root: root, Comments: map[*Node]*Comments{
		root.Out[0].Out[1]: {Line: "# about c"},
	}}
	want := "a b\n  c # about c\n"
	if out := d.Text(); out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestCompact(t *testing.T) {
	long := strings.Repeat("x", 30)
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"arguments", "rule\n  allow\n  any\n", "rule allow any\n"},
		{"single value", "port\n  8080\n", "port 8080\n"},
		{"record", "S1\n  value\n    1.0\n  class\n    asil\n", "S1\n  value 1.0\n  class asil\n"},
		{"run stops at a child with children", "a\n  b\n  c\n    d\n      e\n  f\n", "a b\n  c\n    d e\n  f\n"},
		{"= is content", "a\n  k=v\n  x\n    =v\n", "a k=v\n  x =v\n"},
		{"quoted pieces", "a\n  \"x y\"\n  \"\"\n", "a \"x y\" \"\"\n"},
		{"too wide", "t\n  " + long + "\n  " + long + "\n  " + long + "\n",
			"t\n  " + long + "\n  " + long + "\n  " + long + "\n"},
		{"one wide value stays inline", "t\n  " + long + long + long + "\n", "t " + long + long + long + "\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tree, err := ParseString(c.in)
			if err != nil {
				t.Fatal(err)
			}
			got, err := tree.Compact()
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
			back, err := ParseString(got)
			if err != nil {
				t.Fatalf("compact output does not parse: %v", err)
			}
			if back.Text() != tree.Text() {
				t.Fatalf("compact output denotes a different tree:\n%s\nvs\n%s", back.Text(), tree.Text())
			}
		})
	}
}

// TestCompactRoundTripNames puts awkward names in every position the compact
// writer distinguishes: head, argument, and the head and argument of a child
// line.
func TestCompactRoundTripNames(t *testing.T) {
	names := []string{"plain", "two words", "", `\`, "#", "#x", "a=b", "k=", "=v", "'q", `"q`, "x-y", "S>=S2", "line\nbreak"}
	for _, a := range names {
		for _, b := range names {
			root := &Node{Out: []*Node{
				{Data: a, Out: []*Node{{Data: b}}},
				{Data: "h", Out: []*Node{{Data: a}, {Data: b}}},
				{Data: "h", Out: []*Node{{Data: "k", Out: []*Node{{Data: b}}}, {Data: a, Out: []*Node{{Data: b}}}}},
			}}
			out, err := root.Compact()
			if err != nil {
				t.Fatalf("%q/%q: %v", a, b, err)
			}
			back, err := ParseString(out)
			if err != nil {
				t.Fatalf("%q/%q: reparse: %v\n%s", a, b, err, out)
			}
			if back.Text() != root.Text() {
				t.Fatalf("%q/%q: round trip differs:\n%s\nvs\n%s", a, b, out, root.Text())
			}
		}
	}
}

func TestCompactControlCharacter(t *testing.T) {
	root := &Node{Out: []*Node{{Data: "a\x00"}}}
	if _, err := root.Compact(); !errors.Is(err, ErrControlCharacter) {
		t.Fatalf("expected ErrControlCharacter, got %v", err)
	}
}

// TestEqualsIsContent checks that `=` never splits a word: OGDL has no pairs.
func TestEqualsIsContent(t *testing.T) {
	lx := NewLexer(strings.NewReader(`k=v q="x y" naïve=1 k=\ k=#x`))
	var got []string
	for {
		tok, err := lx.NextToken()
		if err != nil {
			t.Fatal(err)
		}
		if tok.Type == TokenEOF {
			break
		}
		if tok.Type != TokenSpace {
			got = append(got, tok.StringType()+":"+tok.Value)
		}
	}
	want := `Word:k=v Word:q="x Word:y" Word:naïve=1 Word:k=\ Word:k=#x`
	if strings.Join(got, " ") != want {
		t.Fatalf("got  %s\nwant %s", strings.Join(got, " "), want)
	}
}
