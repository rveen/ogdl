package ogdl

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompatBuild(t *testing.T) {
	g := New(nil)
	a := g.Add("a")
	a.Add("b")
	a.Add(42)
	c := New("c")
	if got := g.Add(c); got != c {
		t.Error("Add(*Node) must append the node itself")
	}
	if g.Add(nil) != nil || (*Node)(nil).Add("x") != nil {
		t.Error("Add of nil, or to nil, must return nil")
	}
	if got := g.Text(); got != "a\n  b\n  42\nc\n" {
		t.Errorf("built tree: %q", got)
	}
	g.AddNodes(FromString("d\ne"))
	if g.Len() != 4 || g.GetAt(3).ThisString() != "e" {
		t.Errorf("AddNodes: %q", g.Text())
	}
}

// Node, GetAt and Lookup return nil where v2's S, N and Get return an empty
// node, and accessors are safe on nil, so v1-style chains work.
func TestCompatLookups(t *testing.T) {
	g := FromString("a\n  b\n    c\nlist\n  x 1\n  x 2\n")
	if g.Node("a") == nil || g.Node("a.b") != nil || g.Node("zz") != nil {
		t.Error("Node looks at direct children only, nil if none")
	}
	if g.Node("zz").Node("y").String() != "" || g.Node("zz").Len() != -1 || g.Node("zz").Text() != "" {
		t.Error("chains through nil must be safe")
	}
	if g.Lookup("a.b").String() != "c" || g.Lookup("list.x{1}").String() != "2" || g.Lookup("nope") != nil || g.Lookup("a..b") != nil {
		t.Error("Lookup")
	}
	if g.GetAt(1).ThisString() != "list" || g.GetAt(5) != nil || g.GetAt(-1) != nil {
		t.Error("GetAt")
	}
	var nilNode *Node
	if nilNode.N(0) == nil || nilNode.S("a") == nil || nilNode.Lookup("a") != nil || nilNode.Binary() == nil {
		t.Error("nil receivers")
	}
}

func TestCompatEdit(t *testing.T) {
	g := FromString("a\n  b\nx\ny\nx\n")
	g.Delete("x")
	if g.Text() != "a\n  b\ny\n" {
		t.Errorf("Delete: %q", g.Text())
	}
	g.Add([]byte{1})
	g.Delete([]byte{1}) // not comparable: no match, no panic
	if g.Len() != 3 {
		t.Errorf("Delete of a []byte removed something: %d", g.Len())
	}
	g.DeleteAt(2)
	g.DeleteAt(9)
	if g.Text() != "a\n  b\ny\n" {
		t.Errorf("DeleteAt: %q", g.Text())
	}
	n := g.Create("a")
	if n.Len() != 0 || g.Node("a") != n {
		t.Error("Create clears an existing child")
	}
	if g.Create("new") == nil || g.NodeOrNew("new") != g.Node("new") || g.NodeOrNew("other") == nil || g.Len() != 4 {
		t.Errorf("Create/NodeOrNew: %q", g.Text())
	}
	g.Clear()
	if g.Len() != 0 {
		t.Error("Clear")
	}
}

func TestCompatCopy(t *testing.T) {
	g := FromString("a\n  b\n")
	c := g.Clone()
	c.Node("a").Add("z")
	if g.Text() != "a\n  b\n" || c.Text() != "a\n  b\n  z\n" {
		t.Error("Clone must be deep")
	}
	h := New(nil)
	h.Copy(g)
	h.Node("a").Add("w")
	if g.Text() != "a\n  b\n" || h.Text() != "a\n  b\n  w\n" {
		t.Error("Copy must copy deeply")
	}
}

func TestCompatValues(t *testing.T) {
	g := FromString("n 42\nt true\ns hello\nempty\n")
	if g.Node("n").Interface() != "42" || string(g.Node("s").Bytes()) != "hello" || g.Node("empty").Bytes() != nil {
		t.Error("Interface/Bytes take the first child")
	}
	if New(nil).Add(New(nil)) == nil {
		t.Error("Add(&Node{})")
	}
	b := New(nil)
	b.Add([]byte{0, 1})
	if !bytes.Equal(New(nil).AddNodes(FromString("x")).Bytes(), []byte("x")) {
		t.Error("Bytes of text")
	}
	if s, err := g.GetString("n"); s != "42" || err != nil {
		t.Errorf("GetString: %q %v", s, err)
	}
	if _, err := g.GetString("nope"); err == nil {
		t.Error("GetString of a missing path")
	}
	if _, err := g.GetString("empty"); err == nil {
		t.Error("GetString of a node without children")
	}
	if s, _ := g.GetString(""); s != "n" {
		t.Errorf("GetString(\"\") is String(), the first child: %q", s)
	}
	if v, err := g.GetBool("t"); !v || err != nil {
		t.Error("GetBool")
	}
	if _, err := g.GetBool("s"); err == nil {
		t.Error("GetBool of a non-boolean")
	}
}

func TestCompatSortShow(t *testing.T) {
	g := FromString("p\n  name c\np\n  name a\np\n  name b\np\n  name a\n")
	g.Sort("name")
	got := []string{}
	for _, c := range g.Out {
		got = append(got, c.Get("name").String())
	}
	if len(got) != 4 || got[0] != "a" || got[1] != "a" || got[2] != "b" || got[3] != "c" {
		t.Errorf("Sort: %v", got)
	}
	if s := FromString("a\n  b\n").Show(); s != "_\n  a\n    b" {
		t.Errorf("Show: %q", s)
	}
}

func TestCompatHtml(t *testing.T) {
	g := FromString("a\n  b\nc\n")
	want := "<a href='/r/a'>a</a><br>\n&nbsp;&nbsp;b<br>\nc<br>"
	if got := g.Html("/r"); got != want {
		t.Errorf("Html:\n got %q\nwant %q", got, want)
	}
	if got := New(nil).AddNodes(FromString("'<b>'")).Html(""); got != "&lt;b&gt;<br>" {
		t.Errorf("Html escapes names: %q", got)
	}
}

func TestCompatReaders(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.ogdl")
	os.WriteFile(f, []byte("a b\n"), 0o644)
	if FromFile(f).Text() != "a\n  b\n" || FromFile(filepath.Join(dir, "none")) != nil {
		t.Error("FromFile")
	}
	if FromString("a\n\tb\n  c\n") != nil || FromBytes([]byte("x")) == nil {
		t.Error("FromString/FromBytes: nil on malformed input")
	}
	b, _ := FromString("a").MarshalBinary()
	bf := filepath.Join(dir, "a.gb")
	os.WriteFile(bf, append(append([]byte{}, b...), b...), 0o644)
	if FromBinary(b).Text() != "a\n" || FromBinary(b[:3]) != nil {
		t.Error("FromBinary")
	}
	data, _ := os.ReadFile(bf)
	r := bufio.NewReader(bytes.NewReader(data))
	if FromBinaryReader(r) == nil || FromBinaryReader(r) == nil || FromBinaryReader(r) != nil {
		t.Error("FromBinaryReader reads a sequence, nil at the end")
	}
	if FromBinaryFile(bf) != nil {
		t.Error("FromBinaryFile: a file with two streams is not one stream")
	}
}

// Copying a tree into one of its own descendants terminates (v1 overflowed
// the stack).
func TestCompatCopyIntoDescendant(t *testing.T) {
	g := FromString("a\n  b\n")
	done := make(chan struct{})
	go func() { g.Node("a").Copy(g); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Copy into a descendant did not finish")
	}
	if g.Node("a").Node("a") == nil {
		t.Errorf("the copy is missing: %q", g.Text())
	}
}
