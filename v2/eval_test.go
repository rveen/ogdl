package ogdl

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// evalDoc is the context of appendix A of the expression RFC.
const evalDoc = `a
  b 1
  c 2
n 4
s abc
t
  x 1
  x 2
list
  10
  20
  30
ptr b
`

func evalTree(t *testing.T) *Node {
	t.Helper()
	g, err := ParseString(evalDoc)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// kindOf names the value kinds of expression RFC section 2.1.
func kindOf(v interface{}) string {
	switch v.(type) {
	case nil:
		return "nil"
	case int64:
		return "integer"
	case float64:
		return "float"
	case string:
		return "string"
	case bool:
		return "boolean"
	case *Node:
		return "node"
	}
	return fmt.Sprintf("%T", v)
}

// TestEvalLiterals covers appendix A.1.
func TestEvalLiterals(t *testing.T) {
	g := evalTree(t)

	cases := []struct {
		in   string
		want string
		kind string
	}{
		{"1", "1", "integer"},
		{"1.5", "1.5", "float"},
		{"1e3", "1000", "float"},
		{"'abc'", "abc", "string"},
		{"true", "true", "boolean"},
		{"1+2*3", "7", "integer"},
		{"(1+2)*3", "9", "integer"},
		{"1-2+3", "2", "integer"},
		{"7/2", "3", "integer"},
		{"7.0/2", "3.5", "float"},
		{"7%2", "1", "integer"},
		{"-3+1", "-2", "integer"},
		{"'ab'+'cd'", "abcd", "string"},
		{"'ab'+1", "ab1", "string"},
	}

	for _, c := range cases {
		v, err := g.EvalString(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.in, err)
			continue
		}
		if got := toString(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.in, got, c.want)
		}
		if got := kindOf(v); got != c.kind {
			t.Errorf("%s has kind %s, want %s", c.in, got, c.kind)
		}
	}
}

// TestEvalPaths covers appendix A.2.
func TestEvalPaths(t *testing.T) {
	g := evalTree(t)

	cases := []struct{ in, want string }{
		{"a.b", "1"},
		{"a.b+a.c", "3"},
		{"n+1", "5"},
		{"s", "abc"},
		{"s+'d'", "abcd"},
		{"list[1]", "20"},
		{"list[1+1]", "30"},
		{"t.x{1}", "2"},
		{"a.(ptr)", "1"},
	}

	for _, c := range cases {
		v, err := g.EvalString(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.in, err)
			continue
		}
		if got := toString(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.in, got, c.want)
		}
	}

	// A path that matches nothing is nil, and that is not an error.
	v, err := g.EvalString("missing")
	if err != nil {
		t.Errorf("missing: unexpected error %v", err)
	}
	if v != nil {
		t.Errorf("missing = %v, want nil", v)
	}
}

// TestEvalLogic covers appendix A.3.
func TestEvalLogic(t *testing.T) {
	g := evalTree(t)

	cases := []struct {
		in   string
		want bool
	}{
		{"2>1", true},
		{"2>=2", true},
		{"n==4", true},
		{"n!=4", false},
		{"s=='abc'", true},
		{"s<'abd'", true},
		{"s>'abd'", false},
		{"a.b==1 && a.c==2", true},
		{"a.b==9 || a.c==2", true},
		{"!(a.b==1)", false},
		{"true && false", false},
		{"true || false", true},
	}

	for _, c := range cases {
		v, err := g.EvalString(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.in, err)
			continue
		}
		b, ok := v.(bool)
		if !ok {
			t.Errorf("%s returned %s, want boolean", c.in, kindOf(v))
			continue
		}
		if b != c.want {
			t.Errorf("%s = %v, want %v", c.in, b, c.want)
		}
	}
}

// TestEvalShortCircuit checks that an unevaluated operand cannot fail, and that
// its calls are not made (Expr RFC 6.4).
func TestEvalShortCircuit(t *testing.T) {
	calls := 0
	g := &Node{Out: []*Node{
		{Data: "flag", Out: []*Node{{Data: false}}},
		{Data: "boom", Out: []*Node{{Data: func() int { calls++; return 1 }}}},
	}}

	v, err := g.EvalString("flag && boom()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != false {
		t.Errorf("got %v, want false", v)
	}
	if calls != 0 {
		t.Errorf("right operand was evaluated: %d calls", calls)
	}

	// And the mirror case for ||.
	if _, err := g.EvalString("!flag || boom()"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 0 {
		t.Errorf("right operand of || was evaluated: %d calls", calls)
	}
}

// TestEvalErrors covers appendix A.4.
func TestEvalErrors(t *testing.T) {
	g := evalTree(t)

	cases := []struct {
		in   string
		want error
	}{
		{"1/0", ErrDivideByZero},
		{"1%0", ErrDivideByZero},
		{"1.5%2", ErrNotANumber},
		{"-s", ErrNotANumber},
		{"true<false", ErrNotComparable},
		{"missing+1", ErrNotANumber},
		{"list[-1]", ErrNegativeIndex},
		{"list['x']", ErrNotANumber},
	}

	for _, c := range cases {
		_, err := g.EvalString(c.in)
		if err == nil {
			t.Errorf("%s: expected %v, got none", c.in, c.want)
			continue
		}
		if !errors.Is(err, c.want) {
			t.Errorf("%s: expected %v, got %v", c.in, c.want, err)
		}
	}
}

// TestEvalAssign covers appendix A.3b.
func TestEvalAssign(t *testing.T) {
	cases := []struct {
		expr string
		want string // the value of the expression
		path string // what to read back
		read string // and what it should be
	}{
		{"a.b = 9", "9", "a.b", "9"},
		{"a.b = 1+2", "3", "a.b", "3"},
		{"x.y.z = 1", "1", "x.y.z", "1"},
		{"a.b += 1", "2", "a.b", "2"},
		{"a.b *= 3", "3", "a.b", "3"},
		{"a.b -= 1", "0", "a.b", "0"},
		{"n = n + 1", "5", "n", "5"},
		{"a.b = a.c", "2", "a.b", "2"},
		{"a.b = 'x'+'y'", "xy", "a.b", "xy"},
		{"t.x{1} = 9", "9", "t.x{1}", "9"},
	}

	for _, c := range cases {
		g := evalTree(t)
		v, err := g.EvalString(c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got := toString(v); got != c.want {
			t.Errorf("%s evaluated to %q, want %q", c.expr, got, c.want)
		}
		if got := g.Get(c.path).String(); got != c.read {
			t.Errorf("%s: %s reads %q, want %q", c.expr, c.path, got, c.read)
		}
	}

	// An index target replaces the item itself, so it reads back with
	// ThisString (Expr RFC 6.5).
	g := evalTree(t)
	if _, err := g.EvalString("list[1] = 99"); err != nil {
		t.Fatal(err)
	}
	if got := g.Get("list[1]").ThisString(); got != "99" {
		t.Errorf("list[1] = %q, want 99", got)
	}
	if got, _ := g.Canonical(); !strings.Contains(got, "list\n  10\n  99\n  30\n") {
		t.Errorf("unexpected tree after list[1] = 99:\n%s", got)
	}

	// Appending one past the end.
	g = evalTree(t)
	if _, err := g.EvalString("list[3] = 99"); err != nil {
		t.Fatal(err)
	}
	if got := g.Get("list[3]").ThisString(); got != "99" {
		t.Errorf("list[3] = %q, want 99", got)
	}

	// Right-associative chaining.
	g = evalTree(t)
	if _, err := g.EvalString("p = q = 7"); err != nil {
		t.Fatal(err)
	}
	if g.Get("p").String() != "7" || g.Get("q").String() != "7" {
		t.Errorf("p = q = 7 gave p=%q q=%q", g.Get("p").String(), g.Get("q").String())
	}
}

func TestEvalAssignErrors(t *testing.T) {
	cases := []struct {
		expr string
		want error
	}{
		{"1 = 2", ErrNotAssignable},
		{"1+1 = 2", ErrNotAssignable},
		{"f() = 2", ErrNotAssignable},
		{"t.x{} = 2", ErrTargetIsSet},
		{"list[9] = 1", ErrTargetIndex},
		{"t.x{5} = 1", ErrTargetMissing},
		{"missing += 1", ErrTargetMissing},
		{"a.b /= 0", ErrDivideByZero},
	}

	for _, c := range cases {
		g := evalTree(t)
		before, _ := g.Canonical()

		_, err := g.EvalString(c.expr)
		if err == nil {
			t.Errorf("%s: expected %v, got none", c.expr, c.want)
			continue
		}
		if !errors.Is(err, c.want) {
			t.Errorf("%s: expected %v, got %v", c.expr, c.want, err)
		}

		// A rejected assignment leaves the tree untouched (Expr RFC 6.6).
		after, _ := g.Canonical()
		if before != after {
			t.Errorf("%s modified the tree despite failing:\n%q\n%q", c.expr, before, after)
		}
	}
}

// Assignment is an effect, so a failing right operand must not write
// (Expr RFC 6.6).
func TestEvalAssignRightOperandFirst(t *testing.T) {
	g := evalTree(t)
	before, _ := g.Canonical()

	if _, err := g.EvalString("a.b = 1/0"); err == nil {
		t.Fatal("expected an error")
	}

	after, _ := g.Canonical()
	if before != after {
		t.Fatalf("the target was written before the right operand failed:\n%q\n%q", before, after)
	}
}

// TestEvalBuiltins covers the built-in members of expression RFC 8.4.
func TestEvalBuiltins(t *testing.T) {
	g := evalTree(t)

	cases := []struct {
		in   string
		want string
		kind string
	}{
		{"a._len", "2", "integer"},     // a has children b and c
		{"list._len", "3", "integer"},  // 10, 20, 30
		{"a.b._len", "1", "integer"},   // b holds one leaf
		{"missing._len", "", "nil"},    // nothing to measure
		{"a.b._string", "1", "string"}, // the text below b
		{"a.b._thisString", "b", "string"},
		{"a.b._name", "b", "string"},
		{"a._name", "a", "string"},
		{"s._string", "abc", "string"},
		{"_len", "6", "integer"}, // the context itself: a n s t list ptr
		{"a._this", "a", "node"},
		{"a._this.b", "1", "string"}, // a path continues through _this
		{"a._len + 1", "3", "integer"},
		{"a.b._raw", "1", "ogdl.HTML"},            // what $a.b would write
		{"a._raw", "b\n  1\nc\n  2", "ogdl.HTML"}, // a subtree renders canonically
	}

	for _, c := range cases {
		v, err := g.EvalString(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.in, err)
			continue
		}
		if got := kindOf(v); got != c.kind {
			t.Errorf("%s has kind %s, want %s", c.in, got, c.kind)
			continue
		}
		if c.kind == "node" {
			continue
		}
		if got := toString(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.in, got, c.want)
		}
	}
}

// A node named like a built-in keeps its own meaning (Expr RFC 8.2).
func TestBuiltinsAreShadowedByTheTree(t *testing.T) {
	g, err := ParseString("a\n  _len mine\n  x 1\n")
	if err != nil {
		t.Fatal(err)
	}
	v, err := g.EvalString("a._len")
	if err != nil {
		t.Fatal(err)
	}
	if got := toString(v); got != "mine" {
		t.Fatalf("a._len = %q, want the node's own value", got)
	}
}

// Built-ins belong to the expression layer, not to strict paths
// (Expr RFC 8.4).
func TestBuiltinsNotInStrictPaths(t *testing.T) {
	g := evalTree(t)

	if got := g.Get("a._len").ThisString(); got != "" {
		t.Errorf(`Get("a._len") resolved a built-in: %q`, got)
	}
	if got := g.Get("a.b._raw").ThisString(); got != "" {
		t.Errorf(`Get("a.b._raw") resolved a built-in: %q`, got)
	}
	// The same name through the expression layer does compute.
	v, err := g.EvalString("a._len")
	if err != nil {
		t.Fatal(err)
	}
	if toString(v) != "2" {
		t.Errorf("a._len = %v, want 2", v)
	}
}

func TestEvalDepthLimit(t *testing.T) {
	g := evalTree(t)
	expr := "1"
	for i := 0; i < 400; i++ {
		expr = "(" + expr + "+1)"
	}
	if _, err := g.EvalString(expr); !errors.Is(err, ErrEvalTooDeep) {
		t.Fatalf("expected ErrEvalTooDeep, got %v", err)
	}
}

// The reference implementation's own test cases, adjusted where this
// specification deliberately differs (Expr RFC 12).
func TestEvalReferenceCases(t *testing.T) {
	g, err := ParseString("a '4'\nb 4\n")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct{ in, want, kind string }{
		{"1-2+3", "2", "integer"},
		{"b+3", "7", "integer"},
		// The reference yields "43" here, because it distinguishes a quoted
		// scalar from an unquoted one. A stream carries no types, so both are
		// text and both are numbers (Expr RFC 12).
		{"a+3", "7", "integer"},
	}

	for _, c := range cases {
		v, err := g.EvalString(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.in, err)
			continue
		}
		if got := toString(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.in, got, c.want)
		}
		if got := kindOf(v); got != c.kind {
			t.Errorf("%s has kind %s, want %s", c.in, got, c.kind)
		}
	}

	g2, _ := ParseString("a b\n")
	for _, in := range []string{"a=='b'", "'a'!='b'", "2>1", "2>=2"} {
		v, err := g2.EvalString(in)
		if err != nil {
			t.Errorf("%s: %v", in, err)
			continue
		}
		if v != true {
			t.Errorf("%s = %v, want true", in, v)
		}
	}
}

// TestEvalFloatCompare covers comparison where either operand is a float
// (Expr RFC 6.3): the comparison is numeric, whichever side the float is on.
func TestEvalFloatCompare(t *testing.T) {
	g, err := ParseString("r 0.75\nh 0.5\n")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		in   string
		want bool
	}{
		{"r > 0.5", true},
		{"r < 0.5", false},
		{"r >= 0.75", true},
		{"r <= 0.75", true},
		{"r <= 0.7", false},
		{"r == 0.75", true},
		{"r != 0.75", false},
		{"h < r", true},
		{"1 < 1.5", true},
		{"2 > 1.5", true},
		{"1.5 >= 2", false},
		{"'0.5' == h", true},
	}

	for _, c := range cases {
		v, err := g.EvalString(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.in, err)
			continue
		}
		if b, ok := v.(bool); !ok || b != c.want {
			t.Errorf("%s = %v, want %v", c.in, v, c.want)
		}
	}
}
