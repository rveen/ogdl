package ogdl

import (
	"errors"
	"strings"
	"testing"
)

func TestFormatName(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"hello", "hello"},
		{"hello world", `"hello world"`},
		{`"foo`, `"\"foo"`},
		{"line one\nline two", `"line one\nline two"`},
		{"a\tb", `"a\tb"`},
		{`\`, `"\\"`},
		{"", `""`},
		{"#", `"#"`},
		{"a#b", "a#b"},
		{`C:\path\to`, `C:\path\to`},
		{"it's", "it's"},
		{"a=b", "a=b"},
		{`k="v"`, `k="v"`},
		{"a=", "a="},
		{"=b", "=b"},
		{"a==b", "a==b"},
		{"S>=S2", "S>=S2"},
		{"x.y=1", "x.y=1"},
	}
	for _, c := range cases {
		got, err := formatName(c.name)
		if err != nil {
			t.Errorf("formatName(%q): unexpected error %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("formatName(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFormatNameControlChar(t *testing.T) {
	if _, err := formatName("a\x00b"); !errors.Is(err, ErrControlCharacter) {
		t.Fatalf("expected ErrControlCharacter, got %v", err)
	}
}

// canonical parses s and returns its canonical serialization.
func canonical(t *testing.T, s string) string {
	t.Helper()
	tree, err := ParseString(s)
	if err != nil {
		t.Fatalf("parse error for %q: %v", s, err)
	}
	out, err := tree.Canonical()
	if err != nil {
		t.Fatalf("serialization error for %q: %v", s, err)
	}
	return out
}

// TestVectors covers Appendix A of the RFC.
func TestVectors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "A.1 head and arguments",
			in:   "a b c\n",
			want: "a\n  b\n  c\n",
		},
		{
			name: "A.2 indented line attaches to the head",
			in:   "a b c\n  d\n",
			want: "a\n  b\n  c\n  d\n",
		},
		{
			name: "A.3 siblings and dedent",
			in:   "a\n  b x\n    c f\n  d\ne\n",
			want: "a\n  b\n    x\n    c\n      f\n  d\ne\n",
		},
		{
			name: "A.4 quoting",
			in:   "a 'hola hola'\nb \"with \\\"quotes\\\"\"\nc \"\"\nd C:\\path\\to\n",
			want: "a\n  \"hola hola\"\nb\n  \"with \\\"quotes\\\"\"\nc\n  \"\"\nd\n  C:\\path\\to\n",
		},
		{
			name: "A.5 multi-line quoted string",
			in:   "text \"line one\n      line two\"\n",
			want: "text\n  \"line one\\nline two\"\n",
		},
		{
			name: "A.6 block",
			in:   "description \\\n  some long text\n    indented further\n  and some more.\n",
			want: "description\n  \"some long text\\n  indented further\\nand some more.\"\n",
		},
		{
			name: "A.6b block with interior blank lines",
			in:   "text \\\n  first paragraph\n          \n  second paragraph\n\nnext\n",
			want: "text\n  \"first paragraph\\n\\nsecond paragraph\"\nnext\n",
		},
		{
			name: "A.7 comments and blank lines",
			in:   "# leading comment\n\na b   # trailing comment\n#\na#b\n",
			want: "a\n  b\na#b\n",
		},
		{
			name: "A.8 duplicate names",
			in:   "a 1\na 2\n",
			want: "a\n  1\na\n  2\n",
		},
		{
			name: "A.10 = is content",
			in:   "w a=b S>=S2 --flag=x a= =b a==b x.y=1 url=http://x?a=b k=\\ \"k=v\"\n",
			want: "w\n  a=b\n  S>=S2\n  --flag=x\n  a=\n  =b\n  a==b\n  x.y=1\n  url=http://x?a=b\n  k=\\\n  k=v\n",
		},
		{
			name: "A.11 a record takes a line per field",
			in:   "S1 value 1.0 class asil\nS2\n  value 2.0\n  class asil\n",
			want: "S1\n  value\n  1.0\n  class\n  asil\nS2\n  value\n    2.0\n  class\n    asil\n",
		},
		{
			name: "A.12 block after arguments",
			in:   "text en \\\n  hello\n",
			want: "text\n  en\n  hello\n",
		},
		{
			name: "A.9 empty document",
			in:   "\n   \n# just a comment\n",
			want: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := canonical(t, c.in); got != c.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, c.want)
			}
		})
	}
}

// TestVectorsIllFormed covers Appendix A.13 and RFC section 11.
func TestVectorsIllFormed(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want error
	}{
		{"dedent to unopened level", "a\n    b\n  c\n", ErrUnopenedDedent},
		{"tab after space", "a\n  b\n\tc\n", ErrInconsistentIndentation},
		{"mixed prefix", "a\n \tb\n", ErrMixedIndentation},
		{"unterminated quoted", "a \"unterminated\n", ErrUnterminatedQuoted},
		{"undefined escape", "a \"bad \\q escape\"\n", ErrInvalidEscape},
		{"control character", "a\x00b\n", ErrControlCharacter},
		{"block not last", "a \\ b\n  x\n", ErrBlockNotLast},
		{"block not indented", "a \\\nb\n", ErrBlockIndentation},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseString(c.in)
			if err == nil {
				t.Fatalf("expected error %v, got none", c.want)
			}
			if !errors.Is(err, c.want) {
				t.Fatalf("expected %v, got %v", c.want, err)
			}
			var se *SyntaxError
			if !errors.As(err, &se) {
				t.Fatalf("expected a *SyntaxError, got %T", err)
			}
			if se.Line < 1 {
				t.Errorf("expected a line number, got %d", se.Line)
			}
		})
	}
}

func TestCanonicalIsIdempotent(t *testing.T) {
	inputs := []string{
		"a b c\n  d\n",
		"network\n  eth0\n    ip 192.168.0.10\n    gw   192.168.0.1\n",
		"description \\\n  some long text\n\n  and some more.\n",
		"a 'hola hola'\n",
	}
	for _, in := range inputs {
		once := canonical(t, in)
		twice := canonical(t, once)
		if once != twice {
			t.Errorf("canonicalization not idempotent for %q:\n%q\n%q", in, once, twice)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	names := []string{
		"plain",
		"hello world",
		`"quoted`,
		`\`,
		"#",
		"line one\nline two",
		"tab\there",
		"",
		"it's",
		`C:\path\to`,
		"a=b",
		"k=",
		"x-1=y",
	}
	for _, name := range names {
		root := &Node{Out: []*Node{{Data: name}}}
		serialized, err := root.Canonical()
		if err != nil {
			t.Fatalf("name=%q: %v", name, err)
		}
		reparsed, err := ParseString(serialized)
		if err != nil {
			t.Fatalf("name=%q: parse error: %v\nserialized: %q", name, err, serialized)
		}
		if got := reparsed.N(0).Data; got != name {
			t.Fatalf("name=%q: round-trip gave %q (serialized %q)", name, got, serialized)
		}
	}
}

func TestBOMIgnored(t *testing.T) {
	tree, err := ParseString("\uFEFFa b\n")
	if err != nil {
		t.Fatal(err)
	}
	if tree.N(0).Data != "a" {
		t.Fatalf("expected a, got %q", tree.N(0).Data)
	}
}

func TestCRLF(t *testing.T) {
	got := canonical(t, "a\r\n  b\r\nc\r")
	want := "a\n  b\nc\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBlankLineWithTabInSpaceStream(t *testing.T) {
	// A blank line indents nothing, so its whitespace is exempt (RFC 4.2).
	if _, err := ParseString("a\n  b\n\t\nc\n"); err != nil {
		t.Fatalf("blank line with a TAB should be accepted: %v", err)
	}
}

func TestBareBlockLine(t *testing.T) {
	tree, err := ParseString("a\n\\\n  text\nb\n")
	if err != nil {
		t.Fatal(err)
	}
	if tree.N(1).Data != "text" {
		t.Fatalf("expected the block at depth 0, got %q", tree.N(1).Data)
	}
	if tree.N(2).Data != "b" {
		t.Fatalf("expected b after the block, got %q", tree.N(2).Data)
	}
}

func TestTabIndentation(t *testing.T) {
	got := canonical(t, "a\n\tb\n\t\tc\n\td\n")
	want := "a\n  b\n    c\n  d\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDeepNestingLimit(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		sb.WriteString(strings.Repeat(" ", i))
		sb.WriteString("a\n")
	}
	p, _ := NewParser(strings.NewReader(sb.String()))
	p.MaxDepth = 10
	if _, err := p.Parse(); !errors.Is(err, ErrTooDeep) {
		t.Fatalf("expected ErrTooDeep, got %v", err)
	}
}

func TestBlockNested(t *testing.T) {
	tree, err := ParseString("root\n  desc \\\n    hello\n    world\nother\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := tree.N(0).N(0).N(0).Data; got != "hello\nworld" {
		t.Fatalf("block content = %q", got)
	}
	if got := tree.N(1).Data; got != "other" {
		t.Fatalf("expected other after the block, got %q", got)
	}
}
