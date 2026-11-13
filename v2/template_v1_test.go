package ogdl

import (
	"errors"
	"fmt"
	"testing"
)

type v1Host struct{}

func (*v1Host) Two(a string, b int64) string  { return fmt.Sprintf("<%s|%d>", a, b) }
func (*v1Host) Node(n *Node, s string) string { return fmt.Sprintf("<%v|%s>", n != nil, s) }
func (*v1Host) Err() (string, error)          { return "", errors.New("boom") }
func (*v1Host) Empty() *Node                  { return New(nil) }
func (*v1Host) Nil() *Node                    { return nil }
func (*v1Host) Count(n *Node) int {
	if n == nil {
		return -1
	}
	return len(n.Out)
}

// TestTemplateV1 checks Template.V1 against the output of ogdl v1.5.0 for the
// same templates (loop variables written for v2).
func TestTemplateV1(t *testing.T) {
	cases := []struct{ tpl, want string }{
		{"[$r.version]", "[]"}, // a leaf renders as its value, not its name
		{"[$r.x1]", "[hello]"},
		{"[$r.list]", "[a\nb]"},
		{"[$r.missing]", "[]"},
		{"$if(r.version=='')E$else N$end", "E"},
		{"$if(r.missing=='')E$else N$end", "E"},
		{"[$h.Two('x')]", "[<x|0>]"}, // missing argument: zero value
		{"[$h.Two(r.x1, r.n)]", "[<hello|3>]"},
		{"[$h.Two(r.version, r.version)]", "[<|0>]"},
		{"a $h.Err() b", "a  b"}, // a failed call renders nothing, and goes on
		{"[$h.Node(r.x1)]", "[<true|>]"},
		{"[$h.Node(r, 'z')]", "[<true|z>]"},
		{"[$h.Node('text', 'z')]", "[<false|z>]"}, // unconvertible: zero value
		{"[$('p'+r.missing+'q')]", "[]"},
		{"$(s='p'+r.missing+'q')[$s]", "[pq]"}, // nil in + is ""
		{"$(x=h.Empty())[$h.Count(x)]$for(i,x)<$i>$end", "[0]"},
		{"$(x=h.Nil())[$for(i,x)<$i>$end]", "[]"},
		{"$(v=7.9 % 5)[$v]", "[2]"}, // % truncates floats
		{"$for(i,r.list)<$i>$end", "<a><b>"},
	}
	for _, c := range cases {
		ctx := FromString("r\n  version\n  x1 hello\n  list\n    a\n    b\n  n 3\n")
		ctx.Set("h", &v1Host{})
		tp, err := ParseTemplateV1(c.tpl)
		if err != nil {
			t.Errorf("%q: %v", c.tpl, err)
			continue
		}
		got, _ := tp.Process(ctx)
		if got != c.want {
			t.Errorf("%q: got %q, want %q", c.tpl, got, c.want)
		}
	}

	// The errors are still reported, after the whole output.
	tp, _ := ParseTemplateV1("a $h.Err() b $h.Err() c")
	ctx := New(nil)
	ctx.Set("h", &v1Host{})
	out, err := tp.Process(ctx)
	if out != "a  b  c" || err == nil {
		t.Errorf("got %q, %v", out, err)
	}

	// Without V1 the first failure stops the template.
	tp, _ = ParseTemplate("a $h.Err() b")
	if out, err := tp.Process(ctx); out != "a " || err == nil {
		t.Errorf("strict: got %q, %v", out, err)
	}
}
