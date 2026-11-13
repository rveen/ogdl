package ogdl

// v1-style helpers.
//
// These carry the names and the behaviour of github.com/rveen/ogdl (v1)
// methods, so that code written against v1 can move to v2 without rewriting
// every call: doc/migration.md lists them. Like v1, they accept a nil
// receiver, and the lookups among them return nil when nothing matches,
// where v2's own Get, N and S return an empty node.

import (
	"bufio"
	"bytes"
	"errors"
	"html"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
)

// New returns a node holding data (v1: ogdl.New).
func New(data interface{}) *Node {
	return &Node{Data: data}
}

// FromString parses s as v1 did, returning nil if it is not well-formed
// OGDL (v1: ogdl.FromString). A line's elements form a chain (`a b c` is
// a → b → c; see Parser.Chain). ParseV1 reports the error; ParseString
// reads the RFC grammar.
func FromString(s string) *Node {
	n, err := ParseV1(strings.NewReader(s))
	if err != nil {
		return nil
	}
	return n
}

// FromBytes is FromString for a byte slice (v1: ogdl.FromBytes).
func FromBytes(b []byte) *Node {
	n, err := ParseV1(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	return n
}

// FromFile parses the file as FromString does, returning nil if it cannot be
// read or is not well-formed OGDL (v1: ogdl.FromFile).
func FromFile(path string) *Node {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	n, err := ParseV1(f)
	if err != nil {
		return nil
	}
	return n
}

// ParseV1 reads an OGDL document as ogdl v1 did: the elements of a line form
// a chain (see Parser.Chain). Everything else follows the RFC, which v1 text
// written for v1 is: v2 rejects malformed input v1 accepted (mixed
// indentation, invalid UTF-8, control characters), and drops comments where
// v1 made some of them into nodes named "#".
func ParseV1(r io.Reader) (*Node, error) {
	p, err := NewParser(r)
	if err != nil {
		return nil, err
	}
	p.Chain = true
	return p.Parse()
}

// FromBinary decodes one binary stream, returning nil if it is malformed
// (v1: ogdl.FromBinary). ParseBinary reports the error.
func FromBinary(b []byte) *Node {
	n, err := ParseBinary(b)
	if err != nil {
		return nil
	}
	return n
}

// FromBinaryReader decodes the next binary stream of r, returning nil at the
// end of the input or on a malformed stream (v1: ogdl.FromBinaryReader and
// FromBinaryBufioReader). Pass a *bufio.Reader to read a sequence of streams;
// any other reader is wrapped in one, which may read ahead.
func FromBinaryReader(r io.Reader) *Node {
	br, ok := r.(io.ByteReader)
	if !ok {
		br = bufio.NewReader(r)
	}
	n, err := ReadBinary(br)
	if err != nil {
		return nil
	}
	return n
}

// FromBinaryFile decodes the binary stream in a file, returning nil if it
// cannot be read or is malformed (v1: ogdl.FromBinaryFile).
func FromBinaryFile(path string) *Node {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return FromBinary(b)
}

// Add appends a child and returns it. A *Node is appended itself; any other
// value becomes the data of a new child. Add on a nil node, or of nil, does
// nothing and returns nil (v1: Graph.Add).
func (n *Node) Add(v interface{}) *Node {
	if n == nil || v == nil {
		return nil
	}
	if c, ok := v.(*Node); ok {
		if c == nil {
			return nil
		}
		n.Out = append(n.Out, c)
		return c
	}
	c := &Node{Data: v}
	n.Out = append(n.Out, c)
	return c
}

// AddNodes appends the children of m (not m itself) and returns n
// (v1: Graph.AddNodes).
func (n *Node) AddNodes(m *Node) *Node {
	if n == nil || m == nil {
		return nil
	}
	n.Out = append(n.Out, m.Out...)
	return n
}

// Node returns the first child whose textual form is name, or nil. It looks
// at direct children only: name is not a path (v1: Graph.Node). S is the
// same, returning an empty node instead of nil.
func (n *Node) Node(name string) *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Out {
		if nameString(c.Data) == name {
			return c
		}
	}
	return nil
}

// NodeOrNew returns the child named name, adding it if there is none
// (v1: Graph.NodeOrNew).
func (n *Node) NodeOrNew(name string) *Node {
	if n == nil {
		return nil
	}
	if c := n.Node(name); c != nil {
		return c
	}
	return n.Add(name)
}

// Create returns the child named name with its children removed, adding it
// if there is none (v1: Graph.Create).
func (n *Node) Create(name string) *Node {
	if n == nil {
		return nil
	}
	if c := n.Node(name); c != nil {
		c.Out = nil
		return c
	}
	return n.Add(name)
}

// Lookup is Get returning nil, not an empty node, when the path matches
// nothing or is malformed: v1's Get. Code that tests v1's Get for nil uses
// Lookup.
func (n *Node) Lookup(path string) *Node {
	if n == nil {
		return nil
	}
	elems, err := parsePath(path)
	if err != nil {
		return nil
	}
	ev := &evaluator{maxDepth: DefaultMaxEvalDepth}
	v, err := ev.evalPath(n, elems)
	if err != nil || v == nil {
		return nil
	}
	if m, ok := v.(*Node); ok {
		return m
	}
	return &Node{Data: v}
}

// Len returns the number of children, or -1 for a nil node (v1: Graph.Len).
func (n *Node) Len() int {
	if n == nil {
		return -1
	}
	return len(n.Out)
}

// GetAt returns the i-th child, or nil (v1: Graph.GetAt). N is the same,
// returning an empty node instead of nil.
func (n *Node) GetAt(i int) *Node {
	if n == nil || i < 0 || i >= len(n.Out) {
		return nil
	}
	return n.Out[i]
}

// Clear removes all children (v1: Graph.Clear).
func (n *Node) Clear() {
	if n != nil {
		n.Out = nil
	}
}

// Delete removes every child whose data equals v (v1: Graph.Delete). Data
// that cannot be compared with == (a []byte, for instance) never matches.
func (n *Node) Delete(v interface{}) {
	if n == nil {
		return
	}
	out := n.Out[:0]
	for _, c := range n.Out {
		if !sameData(c.Data, v) {
			out = append(out, c)
		}
	}
	// Do not keep the removed nodes reachable through the old array.
	for i := len(out); i < len(n.Out); i++ {
		n.Out[i] = nil
	}
	n.Out = out
}

func sameData(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == b
	}
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta != tb || !ta.Comparable() {
		return false
	}
	return a == b
}

// DeleteAt removes the i-th child, if there is one (v1: Graph.DeleteAt).
func (n *Node) DeleteAt(i int) {
	if n == nil || i < 0 || i >= len(n.Out) {
		return
	}
	copy(n.Out[i:], n.Out[i+1:])
	n.Out[len(n.Out)-1] = nil
	n.Out = n.Out[:len(n.Out)-1]
}

// Clone returns a deep copy of the tree; the data values themselves are
// shared (v1: Graph.Clone).
func (n *Node) Clone() *Node {
	if n == nil {
		return nil
	}
	c := &Node{Data: n.Data}
	if len(n.Out) > 0 {
		c.Out = make([]*Node, len(n.Out))
		for i, m := range n.Out {
			c.Out[i] = m.Clone()
		}
	}
	return c
}

// Copy appends a deep copy of the children of m (v1: Graph.Copy).
func (n *Node) Copy(m *Node) {
	if n == nil || m == nil {
		return
	}
	for _, c := range m.Out {
		n.Out = append(n.Out, c.Clone())
	}
}

// Interface returns the data of the first child, or nil (v1:
// Graph.Interface). Like String, it describes the node by its first child.
func (n *Node) Interface() interface{} {
	if n == nil || len(n.Out) == 0 {
		return nil
	}
	return n.Out[0].Data
}

// Bytes returns the data of the first child as bytes: a []byte as it is,
// anything else as its textual form; nil if there is no child (v1:
// Graph.Bytes).
func (n *Node) Bytes() []byte {
	v := n.Interface()
	if v == nil {
		return nil
	}
	if b, ok := v.([]byte); ok {
		return b
	}
	return []byte(nameString(v))
}

// GetString returns the textual form of the first child of the node at
// path; with an empty path, String(). It reports a path that matches
// nothing, or a node with no children (v1: Graph.GetString).
func (n *Node) GetString(path string) (string, error) {
	if path == "" {
		return n.String(), nil
	}
	m := n.Lookup(path)
	if m == nil {
		return "", errors.New("ogdl: not found: " + path)
	}
	if len(m.Out) == 0 {
		return "", errors.New("ogdl: no value at " + path)
	}
	return nameString(m.Out[0].Data), nil
}

// GetBool is GetString for a boolean: "true" or "false" (v1: Graph.GetBool).
func (n *Node) GetBool(path string) (bool, error) {
	s, err := n.GetString(path)
	if err != nil {
		return false, err
	}
	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, errors.New("ogdl: not a boolean: " + s)
}

// Sort orders the children by the textual form of the node each has at
// path (v1: Graph.Sort). Like v1 it uses sort.Sort, which is not stable:
// children with equal keys come out in the order v1 gives them.
func (n *Node) Sort(path string) {
	if n == nil {
		return
	}
	sort.Sort(byPath{path, n.Out})
}

type byPath struct {
	path string
	out  []*Node
}

func (s byPath) Len() int      { return len(s.out) }
func (s byPath) Swap(i, j int) { s.out[i], s.out[j] = s.out[j], s.out[i] }
func (s byPath) Less(i, j int) bool {
	return s.out[i].Get(s.path).String() < s.out[j].Get(s.path).String()
}

// Show returns the canonical text of the tree including the node itself, a
// node without data shown as "_", with no final line break: a debugging aid
// (v1: Graph.Show).
func (n *Node) Show() string {
	if n == nil {
		return ""
	}
	top := n
	if n.Data == nil {
		top = &Node{Data: "_", Out: n.Out}
	}
	return strings.TrimSuffix((&Node{Out: []*Node{top}}).Text(), "\n")
}

// Html renders the children as lines of HTML, indented with &nbsp;, each
// node with children a link to path/its/names (v1: Graph.Html). Unlike v1,
// names are HTML-escaped.
func (n *Node) Html(path string) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range n.Out {
		c.html(0, &b, path)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func (n *Node) html(level int, b *strings.Builder, path string) {
	sp := strings.Repeat("&nbsp;&nbsp;", level)
	s := nameString(n.Data)
	if s == "" {
		s = "_"
	}
	if len(n.Out) > 0 {
		b.WriteString("<a href='" + html.EscapeString(path+"/"+s) + "'>")
	}
	if strings.ContainsAny(s, "\n\r \t'\",()") {
		if level > 0 {
			b.WriteString(sp)
			b.WriteByte('"')
		}
		t := strings.ReplaceAll(s, "\r", "")
		if level > 0 {
			t = strings.ReplaceAll(t, `"`, `\"`)
		}
		t = html.EscapeString(t)
		b.WriteString(strings.ReplaceAll(t, "\n", "\n"+sp))
		if level > 0 {
			b.WriteByte('"')
		}
		if len(n.Out) > 0 {
			b.WriteString("</a>")
		}
	} else {
		b.WriteString(sp)
		b.WriteString(html.EscapeString(s))
		if len(n.Out) > 0 {
			b.WriteString("</a>")
		}
	}
	b.WriteString("<br>\n")
	for _, c := range n.Out {
		c.html(level+1, b, path+"/"+s)
	}
}
