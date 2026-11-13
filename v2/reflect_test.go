package ogdl

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// host is the test subject for the Go interop rules of expression RFC 8.
type host struct {
	Name   string
	Count  int
	hidden string
}

func (h *host) Add(a, b int) int        { return a + b }
func (h *host) Greet(who string) string { return "hello " + who }
func (h *host) Fail() (int, error)      { return 0, errors.New("boom") }
func (h *host) Ok() (int, error)        { return 7, nil }
func (h *host) Nothing()                {}
func (h *host) Panics() int             { panic("host exploded") }
func (h *host) Sum(xs ...int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}
func (h *host) secret() string        { return h.hidden }
func (h *host) Upper(s string) string { return strings.ToUpper(s) }

func hostTree() (*Node, *host) {
	h := &host{Name: "ada", Count: 3, hidden: "shh"}
	g := &Node{Out: []*Node{
		{Data: "obj", Out: []*Node{{Data: h}}},
		{Data: "fn", Out: []*Node{{Data: func(x int) int { return x * 2 }}}},
		{Data: "cat", Out: []*Node{{Data: func(a, b string) string { return a + b }}}},
		{Data: "m", Out: []*Node{{Data: map[string]int{"one": 1, "two": 2}}}},
		{Data: "sl", Out: []*Node{{Data: []string{"x", "y", "z"}}}},
	}}
	return g, h
}

// TestHostValues covers appendix A.5.
func TestHostValues(t *testing.T) {
	g, _ := hostTree()

	cases := []struct{ in, want string }{
		{"fn(2)", "4"},
		{"fn(1+2)", "6"},
		{"obj.Add(2+3, 1)", "6"},
		{"obj.Greet('world')", "hello world"},
		{"obj.Name", "ada"},
		{"obj.Count", "3"},
		{"obj.Count+1", "4"},
		{"obj.Ok()", "7"},
		{"obj.Sum(1,2,3)", "6"},
		{"obj.Sum()", "0"},
		{"cat('a','b')", "ab"},
		{"m.one", "1"},
		{"sl[1]", "y"},
		{"obj.Upper('abc')", "ABC"},
		// A string argument is converted to the parameter type.
		{"obj.Add('2','3')", "5"},
		// A value reached by reflection can be used as an operand.
		{"obj.Greet('x')+'!'", "hello x!"},
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
}

func TestHostErrors(t *testing.T) {
	g, _ := hostTree()

	cases := []struct {
		in   string
		want error
	}{
		{"obj.Add(1)", ErrArgCount},
		{"obj.Add(1,2,3)", ErrArgCount},
		{"obj.Name(1)", ErrNotCallable},
		{"obj.Greet(1,2)", ErrArgCount},
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

	// A returned error is the expression's error.
	if _, err := g.EvalString("obj.Fail()"); err == nil || err.Error() != "boom" {
		t.Errorf("obj.Fail(): expected the returned error, got %v", err)
	}

	// A panic in host code is reported, not propagated.
	_, err := g.EvalString("obj.Panics()")
	if err == nil || !strings.Contains(err.Error(), "panic in call") {
		t.Errorf("obj.Panics(): expected a recovered panic, got %v", err)
	}

	// A method with no results yields nil.
	v, err := g.EvalString("obj.Nothing()")
	if err != nil || v != nil {
		t.Errorf("obj.Nothing() = %v, %v; want nil, nil", v, err)
	}
}

// Unexported members MUST stay unreachable (Expr RFC 8.2).
func TestHostUnexported(t *testing.T) {
	g, _ := hostTree()

	for _, in := range []string{"obj.hidden", "obj.secret"} {
		v, err := g.EvalString(in)
		if err != nil {
			t.Errorf("%s: %v", in, err)
		}
		if v != nil {
			t.Errorf("%s reached an unexported member: %v", in, v)
		}
	}
}

// The tree shadows the host value, so adding a method cannot change the meaning
// of an expression that already resolved through the tree (Expr RFC 8.2).
func TestTreeShadowsHost(t *testing.T) {
	h := &host{Name: "ada"}
	g := &Node{Out: []*Node{
		{Data: "obj", Out: []*Node{
			{Data: h},
			{Data: "Name", Out: []*Node{{Data: "from the tree"}}},
		}},
	}}

	v, err := g.EvalString("obj.Name")
	if err != nil {
		t.Fatal(err)
	}
	if got := toString(v); got != "from the tree" {
		t.Fatalf("obj.Name = %q, want the tree's value", got)
	}
}

// A missing member is nil, not an error, exactly as a missing node is.
func TestHostMissingMember(t *testing.T) {
	g, _ := hostTree()
	v, err := g.EvalString("obj.Missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != nil {
		t.Fatalf("obj.Missing = %v, want nil", v)
	}
}

func TestHostNodeArgument(t *testing.T) {
	// A parameter of node type receives the node itself (Expr RFC 8.3).
	g := &Node{Out: []*Node{
		{Data: "count", Out: []*Node{{Data: func(n *Node) int { return len(n.Out) }}}},
		{Data: "list", Out: []*Node{{Data: "a"}, {Data: "b"}}},
	}}

	v, err := g.EvalString("count(list)")
	if err != nil {
		t.Fatal(err)
	}
	if toString(v) != "2" {
		t.Fatalf("count(list) = %v, want 2", v)
	}
}

func TestConvertArgRejectsMismatch(t *testing.T) {
	// A string that is not a number must not be passed as a zero int.
	g := &Node{Out: []*Node{
		{Data: "f", Out: []*Node{{Data: func(x int) int { return x }}}},
	}}

	_, err := g.EvalString("f('abc')")
	if !errors.Is(err, ErrIncompatibleType) {
		t.Fatalf("expected ErrIncompatibleType, got %v", err)
	}
}

// TestConvertArg covers argument conversion to each parameter kind (Expr RFC
// 8.3), and the conversions that must fail rather than pass a zero value.
func TestConvertArg(t *testing.T) {
	g := &Node{}
	g.Set("u", func(x uint) uint { return x })
	g.Set("i8", func(x int8) int8 { return x })
	g.Set("f32", func(x float32) float32 { return x })
	g.Set("b", func(x bool) bool { return x })
	g.Set("s", func(x string) string { return x + "!" })
	g.Set("any", func(x interface{}) string { return kindOf(x) })
	g.Set("str", func(x fmt.Stringer) string { return x.String() })
	g.Set("n", "7")

	ok := []struct{ in, want string }{
		{"u(3)", "3"},
		{"u(n)", "7"},
		{"i8(-128)", "-128"},
		{"i8(2.0)", "2"},
		{"u('3')", "3"},
		{"f32(1.5)", "1.5"},
		{"f32(2)", "2"},
		{"b('true')", "true"},
		{"b(1 > 2)", "false"},
		{"s(42)", "42!"},
		{"s(n)", "7!"},
		{"any(1)", kindOf(int64(1))},
		{"any(missing)", kindOf(nil)},
	}
	for _, c := range ok {
		v, err := g.EvalString(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.in, err)
			continue
		}
		if got := toString(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.in, got, c.want)
		}
	}

	bad := []string{
		"u(-1)",      // negative to unsigned
		"u('x')",     // not a number
		"i8(128)",    // overflow
		"f32('x')",   // not a number
		"b('maybe')", // not a boolean
		"str('a')",   // does not implement the interface
	}
	for _, in := range bad {
		if _, err := g.EvalString(in); !errors.Is(err, ErrIncompatibleType) {
			t.Errorf("%s: expected ErrIncompatibleType, got %v", in, err)
		}
	}
}
