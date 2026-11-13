package ogdl

import (
	"errors"
	"strings"
	"testing"
)

// tmplDoc is the context of appendix A of the template RFC.
const tmplDoc = `user
  name Jenny
  admin true
b
  1
  2
cart
  item
    name Widget
    price 10
  item
    name Gadget
    price 20
n 4
`

func tmplTree(t *testing.T) *Node {
	t.Helper()
	g, err := ParseString(tmplDoc)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func runTmpl(t *testing.T, ctx *Node, tpl string) (string, error) {
	t.Helper()
	return ProcessString(tpl, ctx)
}

// TestTemplateSubstitution covers appendix A.1.
func TestTemplateSubstitution(t *testing.T) {
	cases := []struct{ tpl, want string }{
		{"Hello, $user.name", "Hello, Jenny"},
		{"${user.name}!", "Jenny!"},
		{"$b", "1\n2"},
		{"$missing", ""},
		{"[$missing]", "[]"},
		{"$n", "4"},
		{"$$5.00", "$5.00"},
		{"$5.00", "$5.00"},
		{"a $ b", "a $ b"},
		{"no variables here", "no variables here"},
		{"", ""},
		{"$user.name and $n", "Jenny and 4"},
		{"$", "$"},
		// The braced form ends the path where text resumes.
		{"${n}th", "4th"},
		// A directive word is only a directive as the whole path.
		{"$user.name$user.admin", "Jennytrue"},
	}

	for _, c := range cases {
		g := tmplTree(t)
		got, err := runTmpl(t, g, c.tpl)
		if err != nil {
			t.Errorf("%q: %v", c.tpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.tpl, got, c.want)
		}
	}
}

// TestTemplateSilent covers appendix A.2.
func TestTemplateSilent(t *testing.T) {
	cases := []struct{ tpl, want string }{
		{"$(x = 1)$x", "1"},
		{"$(x = n * 2)$x", "8"},
		{"$(x = 1)", ""},
		{"$(user.name = 'Ada')$user.name", "Ada"},
	}

	for _, c := range cases {
		g := tmplTree(t)
		got, err := runTmpl(t, g, c.tpl)
		if err != nil {
			t.Errorf("%q: %v", c.tpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.tpl, got, c.want)
		}
	}
}

// TestTemplateConditionals covers appendix A.3.
func TestTemplateConditionals(t *testing.T) {
	cases := []struct{ tpl, want string }{
		{"$if(user.admin)yes$end", "yes"},
		{"$if(missing)yes$end", ""},
		{"$if(missing)yes$else no$end", " no"},
		{"$if(n>10)a$elseif(n>3)b$else c$end", "b"},
		{"$if(n>10)a$elseif(n>100)b$else c$end", " c"},
		{"$if(n>1)$if(n>3)deep$end$end", "deep"},
		{"$if(n==4)four$end", "four"},
		{"before $if(n>1)yes$end after", "before yes after"},
		// Two chains at the same level do not interfere.
		{"$if(n>10)a$else A$end$if(n>1)b$else B$end", " Ab"},
	}

	for _, c := range cases {
		g := tmplTree(t)
		got, err := runTmpl(t, g, c.tpl)
		if err != nil {
			t.Errorf("%q: %v", c.tpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.tpl, got, c.want)
		}
	}
}

// TestTemplateLoops covers appendix A.4.
func TestTemplateLoops(t *testing.T) {
	cases := []struct{ tpl, want string }{
		{"$for(x, b)[$x]$end", "[1][2]"},
		{"$for(x, cart)$x.name $end", "Widget Gadget "},
		{"$for(x, cart)$x._name $end", "item item "},
		{"$for(x, missing)never$end", ""},
		{"$for(x, user.name)[$x]$end", "[Jenny]"},
		{"$for(x, b[0])[$x]$end", ""},
		{"$for(x, b)$x$break$end", "1"},
		{"$for(x, b)$for(y, b)$x$y$end$end", "11122122"},
		{"$for(x, cart)$if(x.price>15)$x.name$end$end", "Gadget"},
		// The variable is the element: a leaf renders its own name, and a
		// selector addresses the element itself.
		{"$for(x, b)[$x._name]$end", "[1][2]"},
		{"$for(x, cart)$x{0}.name $x{1}.name|$end", "Widget |Gadget |"},
		{"$for(x, cart)$for(y, x{})$y.name $end$end", "Widget Gadget "},
		// Nested loops see the outer element, not a wrapper.
		{"$for(x, cart)$for(y, x)$y._name=$y $end$end", "name=Widget price=10 name=Gadget price=20 "},
	}

	for _, c := range cases {
		g := tmplTree(t)
		got, err := runTmpl(t, g, c.tpl)
		if err != nil {
			t.Errorf("%q: %v", c.tpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.tpl, got, c.want)
		}
	}
}

// TestTemplateScoping covers appendix A.5: a loop leaves the context as it was.
func TestTemplateScoping(t *testing.T) {
	cases := []struct{ tpl, want string }{
		{"$for(n, b)$n$end$n", "124"},
		{"$for(x, b)$x$end$x", "12"},
		{"$for(n, b)[$n $_this.n]$end", "[1 4][2 4]"},
	}

	for _, c := range cases {
		g := tmplTree(t)
		got, err := runTmpl(t, g, c.tpl)
		if err != nil {
			t.Errorf("%q: %v", c.tpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.tpl, got, c.want)
		}
	}

	// The context itself must be unchanged after processing.
	g := tmplTree(t)
	before, _ := g.Canonical()
	if _, err := runTmpl(t, g, "$for(x, b)$x$end"); err != nil {
		t.Fatal(err)
	}
	after, _ := g.Canonical()
	if before != after {
		t.Errorf("the loop modified the context:\n%q\n%q", before, after)
	}

	// Only the loop variable is scoped: what the body does to the rest of the
	// context survives the loop (Template RFC 3, 5.2).
	g = tmplTree(t)
	out, err := runTmpl(t, g, "$for(x, b)$(seen = x)$end$seen")
	if err != nil {
		t.Fatal(err)
	}
	if out != "2" {
		t.Errorf("an assignment inside the loop was lost: got %q, want 2", out)
	}

	// And after a break, too.
	g = tmplTree(t)
	before, _ = g.Canonical()
	if _, err := runTmpl(t, g, "$for(n, b)$n$break$end"); err != nil {
		t.Fatal(err)
	}
	after, _ = g.Canonical()
	if before != after {
		t.Errorf("break left the context modified:\n%q\n%q", before, after)
	}

	// The variable is never written into the tree: during the loop the
	// context still has its four children, and a shadowed node is reachable
	// through _this.
	g = tmplTree(t)
	out, err = runTmpl(t, g, "$for(n, b)[$_this._len $n $_this.n]$end")
	if err != nil {
		t.Fatal(err)
	}
	if want := "[4 1 4][4 2 4]"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// The loop variable is the element, so assigning through it writes into the
// element (Template RFC 5.2).
func TestTemplateLoopAssign(t *testing.T) {
	g := tmplTree(t)
	out, err := runTmpl(t, g, "$for(x, cart)$(x.price *= 2)$(x.seen = 1)$end$for(x, cart)$x.price/$x.seen $end")
	if err != nil {
		t.Fatal(err)
	}
	if want := "20/1 40/1 "; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
	if got := g.Get("cart.item{1}.price").String(); got != "40" {
		t.Errorf("cart.item{1}.price = %q, want 40", got)
	}

	// Through a selector on the variable, too; occurrence 1 does not exist.
	g = tmplTree(t)
	if _, err := runTmpl(t, g, "$for(x, cart)$(x{0}.price = 0)$end"); err != nil {
		t.Fatal(err)
	}
	if got := g.Get("cart.item{1}.price").String(); got != "0" {
		t.Errorf("selector assignment: price = %q, want 0", got)
	}
	if _, err := runTmpl(t, g, "$for(x, cart)$(x{1}.price = 0)$end"); !errors.Is(err, ErrTargetMissing) {
		t.Errorf("x{1} = ...: got %v, want ErrTargetMissing", err)
	}

	// A variable is bound only at the start of a path: a child named x of
	// another node is still that child.
	g = tmplTree(t)
	out, err = runTmpl(t, g, "$for(name, b)$user.name $end")
	if err != nil {
		t.Fatal(err)
	}
	if out != "Jenny Jenny " {
		t.Errorf("got %q, want %q", out, "Jenny Jenny ")
	}
}

// A host element is bound as itself: its fields are reachable directly.
func TestTemplateLoopHostElements(t *testing.T) {
	type row struct{ ID, Name string }
	g := &Node{}
	g.Set("rows", []row{{"1", "a"}, {"2", "b"}})
	out, err := ProcessString("$for(r, rows)[$r.ID/$r.Name]$end", g)
	if err != nil {
		t.Fatal(err)
	}
	if want := "[1/a][2/b]"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// TestTemplateErrors covers appendix A.6.
func TestTemplateErrors(t *testing.T) {
	cases := []struct {
		tpl  string
		want error
	}{
		{"$if(n>1)x", ErrMissingEnd},
		{"$end", ErrUnexpectedEnd},
		{"$else x$end", ErrUnexpectedElse},
		{"$break", ErrBreakOutsideLoop},
		{"$for(n)x$end", ErrBadFor},
		{"$for(1, b)x$end", ErrBadFor},
		{"$(1/0)", ErrDivideByZero},
		{"${user.name", ErrUnterminatedTpl},
		{"$if(n>1)a$else b$else c$end", ErrDoubleElse},
		{"$for(x, b)x", ErrMissingEnd},
	}

	for _, c := range cases {
		g := tmplTree(t)
		_, err := runTmpl(t, g, c.tpl)
		if err == nil {
			t.Errorf("%q: expected %v, got none", c.tpl, c.want)
			continue
		}
		if !errors.Is(err, c.want) {
			t.Errorf("%q: expected %v, got %v", c.tpl, c.want, err)
		}
	}
}

// Errors are reported rather than swallowed (Template RFC 6, 10).
func TestTemplateReportsEvalErrors(t *testing.T) {
	g := tmplTree(t)
	if _, err := runTmpl(t, g, "total: $(1/0)"); err == nil {
		t.Error("a failed expression must be reported")
	}
	// But a missing path is not an error.
	out, err := runTmpl(t, g, "[$missing.deeper]")
	if err != nil {
		t.Errorf("a missing path must not be an error: %v", err)
	}
	if out != "[]" {
		t.Errorf("got %q, want %q", out, "[]")
	}
}

func TestTemplateCallsHost(t *testing.T) {
	g := tmplTree(t)
	g.Set("upper", func(s string) string {
		out := []rune(s)
		for i, r := range out {
			if r >= 'a' && r <= 'z' {
				out[i] = r - 32
			}
		}
		return string(out)
	})

	got, err := runTmpl(t, g, "$upper(user.name)")
	if err != nil {
		t.Fatal(err)
	}
	if got != "JENNY" {
		t.Fatalf("got %q, want JENNY", got)
	}
}

// Built-in members are available in templates (Expr RFC 8.4), which is what the
// reference's own template examples use them for.
func TestTemplateBuiltins(t *testing.T) {
	cases := []struct{ tpl, want string }{
		{"$b._len", "2"},
		{"$cart._len items", "2 items"},
		{"$for(x, cart)$x._name $end", "item item "},
		{"$for(x, cart)$x._thisString $end", "item item "},
		{"$user.name._string", "Jenny"},
		{"$if(cart._len > 1)many$end", "many"},
		{"$for(x, cart)[$x._len]$end", "[2][2]"},
	}

	for _, c := range cases {
		g := tmplTree(t)
		got, err := runTmpl(t, g, c.tpl)
		if err != nil {
			t.Errorf("%q: %v", c.tpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.tpl, got, c.want)
		}
	}
}

func TestTemplateTextIsCopiedUnchanged(t *testing.T) {
	g := tmplTree(t)
	tpl := "line one\r\n  indented\ttab\n\nblank above $n\n"
	want := "line one\r\n  indented\ttab\n\nblank above 4\n"
	got, err := runTmpl(t, g, tpl)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("text was not copied unchanged:\n%q\n%q", got, want)
	}
}

func TestTemplateDepthLimit(t *testing.T) {
	g := tmplTree(t)
	const depth = 201
	tpl := strings.Repeat("$if(n>1)", depth) + "x" + strings.Repeat("$end", depth)

	tp, err := ParseTemplate(tpl)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tp.Process(g); !errors.Is(err, ErrEvalTooDeep) {
		t.Fatalf("expected a depth error, got %v", err)
	}
}

// TestTemplateLoopsOverHost covers iteration of host values (Template RFC 5.1).
// The length comes from the value: a nil element is iterated like any other.
func TestTemplateLoopsOverHost(t *testing.T) {
	cases := []struct {
		name string
		v    interface{}
		want string
	}{
		{"slice", []string{"a", "b"}, "[a][b]"},
		{"nil element", []interface{}{"a", nil, "c"}, "[a][][c]"},
		{"leading nil", []interface{}{nil, "b"}, "[][b]"},
		{"array", [2]int{1, 2}, "[1][2]"},
		{"pointer to slice", &[]string{"x"}, "[x]"},
		{"nil pointer", (*[]string)(nil), ""},
		{"empty slice", []string{}, ""},
	}

	for _, c := range cases {
		g := &Node{}
		g.Set("list", c.v)
		got, err := runTmpl(t, g, "$for(x, list)[$x]$end")
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestTemplateLoopShortLists checks that a node iterates its children however
// few there are (Template RFC 5.1): operand simplification must not turn a
// one-element list into a string, nor an empty list into its own name.
func TestTemplateLoopShortLists(t *testing.T) {
	g, err := ParseString("one\n  x\nnone\nnested\n  list\n    y\ntwo\n  a\n  b\n")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ tpl, want string }{
		{"$for(i, one)[$i]$end", "[x]"},
		{"$for(i, none)[$i]$end", ""},
		{"$for(i, nested.list)[$i]$end", "[y]"},
		{"$for(i, two)[$i]$end", "[a][b]"},
		// A set with a single leaf member iterates that member.
		{"$for(i, one.x{})[$i]$end", "[x]"},
		// A missing path is nil: no iterations, no error.
		{"$for(i, missing)[$i]$end", ""},
	}
	for _, c := range cases {
		got, err := runTmpl(t, g, c.tpl)
		if err != nil {
			t.Errorf("%q: %v", c.tpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.tpl, got, c.want)
		}
	}
}

// TestTemplateLoopNotIterable checks that a value that is neither a node, nil,
// nor a host slice or array is an error (Template RFC 5.1) — a string included.
func TestTemplateLoopNotIterable(t *testing.T) {
	for _, tpl := range []string{"$for(x, 5)$x$end", "$for(x, 'abc')$x$end", "$for(x, m)$x$end"} {
		g := &Node{}
		g.Set("m", map[string]int{"a": 1})
		if _, err := runTmpl(t, g, tpl); !errors.Is(err, ErrNotIterable) {
			t.Errorf("%q: expected ErrNotIterable, got %v", tpl, err)
		}
	}
}

// TestTemplateEscapeHTML covers the HTML escaping mode (Template RFC 4.1,
// appendix A.7):
// substitutions are escaped, template text is not, and a host HTML value is
// written unchanged.
func TestTemplateEscapeHTML(t *testing.T) {
	cases := []struct{ tpl, want string }{
		{"<p>$user.name</p>", "<p>Jenny</p>"},
		{"$(x = '<b>')$x", "&lt;b&gt;"},
		{"$(x = 'a&b')$x", "a&amp;b"},
		{"$(x = 'a&b')${x}", "a&amp;b"},
		{`$(x = 'a"b')<a title="$x">`, `<a title="a&#34;b">`},
		{`$(x = "it's")$x`, "it&#39;s"},
		{"$for(i, tags)<li>$i</li>$end", "<li>&lt;script&gt;</li><li>a &lt; b</li>"},
		{"$b", "1\n2"},
		{"$safe", "<b>ok</b>"},
		{"$raw('<i>x</i>')", "<i>x</i>"},
		{"$raw(tags[0])", "<script>"},
		{"$up('<i>')", "&lt;I&gt;"},
	}

	for _, c := range cases {
		g := tmplTree(t)
		g.Set("tags", nil)
		g.Get("tags").Out = []*Node{{Data: "<script>"}, {Data: "a < b"}}
		g.Set("safe", HTML("<b>ok</b>"))
		g.Set("raw", func(s string) HTML { return HTML(s) })
		g.Set("up", strings.ToUpper)

		got, err := ProcessHTMLString(c.tpl, g)
		if err != nil {
			t.Errorf("%q: %v", c.tpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.tpl, got, c.want)
		}
	}
}

// TestTemplateEscapeHTMLIsOptIn checks that a plain template writes markup in
// values unchanged, and that the mode can be set on a parsed template.
func TestTemplateEscapeHTMLIsOptIn(t *testing.T) {
	g := tmplTree(t)
	g.Set("x", "<b>")

	tpl, err := ParseTemplate("$x")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := tpl.Process(g); got != "<b>" {
		t.Errorf("plain template = %q, want <b>", got)
	}
	tpl.EscapeHTML = true
	if got, _ := tpl.Process(g); got != "&lt;b&gt;" {
		t.Errorf("escaping template = %q, want &lt;b&gt;", got)
	}

	if _, err := ParseHTMLTemplate("$if(x)"); !errors.Is(err, ErrMissingEnd) {
		t.Errorf("ParseHTMLTemplate: expected ErrMissingEnd, got %v", err)
	}
}

// TestTemplateRaw covers the built-in _raw (Expr RFC 8.4, Template RFC 4.1):
// in an escaping template it writes what a substitution would write, but
// unescaped; in a plain template it changes nothing.
func TestTemplateRaw(t *testing.T) {
	ctx := func() *Node {
		g := tmplTree(t)
		g.Set("body", "<p>hi</p>")
		g.Set("tags", nil)
		g.Get("tags").Out = []*Node{{Data: "<b>"}, {Data: "<i>"}}
		return g
	}

	html := []struct{ tpl, want string }{
		{"$body", "&lt;p&gt;hi&lt;/p&gt;"},
		{"$body._raw", "<p>hi</p>"},
		{"${body._raw}", "<p>hi</p>"},
		{"$tags._raw", "<b>\n<i>"},
		{"$for(x, tags)$x._raw$end", "<b><i>"},
		{"$missing._raw", ""},
		{"$(x = '<b>')$x._raw", "<b>"},
	}
	for _, c := range html {
		got, err := ProcessHTMLString(c.tpl, ctx())
		if err != nil {
			t.Errorf("%q: %v", c.tpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.tpl, got, c.want)
		}
	}

	// Without escaping, _raw is the same text a plain substitution writes.
	for _, tpl := range []string{"$body", "$body._raw"} {
		if got, _ := ProcessString(tpl, ctx()); got != "<p>hi</p>" {
			t.Errorf("plain %q = %q, want <p>hi</p>", tpl, got)
		}
	}

	// A node named _raw is the document's own, and stays escaped: data cannot
	// switch escaping off (Expr RFC 8.2).
	g, err := ParseString("body\n  _raw '<script>'\n")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := ProcessHTMLString("$body._raw", g); got != "&lt;script&gt;" {
		t.Errorf("shadowed _raw = %q, want it escaped", got)
	}
}
