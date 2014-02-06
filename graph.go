// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
)

// Graph is a node with outgoing pointers to other Graph objects.
// It is implemented as a named list.
type Graph struct {
	This interface{}
	Out  []*Graph
}

// NewGraph creates a Graph instance with the given name.
// At this stage it is a single node without outgoing edges.
func NewGraph(n interface{}) *Graph {
	return &Graph{n, nil}
}

// NilGraph returns a pointer to a 'null' Graph, also called transparent
// node.
func NilGraph() *Graph {
	return &Graph{}
}

// IsNil returns true is this node has no content, i.e, is a transparent node.
func (g *Graph) IsNil() bool {
	if g.This == nil {
		return true
	}
	return false
}

// Len returns the number of subnodes (outgoing edges, out degree) of this node.
func (g *Graph) Len() int {
	if g == nil {
		return -1
	}
	return len(g.Out)
}

// Type returns the name of the native type contained in the current node.
func (g *Graph) Type() string {
	return reflect.TypeOf(g.This).String()
}

// Depth returns the depth of the graph if it is a tree, or -1 if it has
// cycles.
//
// TODO: Cycles are inferred if level>100, but nodes traversed are not
// remembered (they should if cycles need to be detected).
func (g *Graph) Depth() int {
	if g.Len() == 0 {
		return 0
	}

	i := 0
	for _, n := range g.Out {
		j := n.Depth()
		if j > i {
			i = j
		}
	}

	if i > 100 {
		return -1
	}
	return i + 1
}

// Equal returns true if the given graph and the receiver graph are equal.
func (g *Graph) Equal(c *Graph) bool {

	if c.This != g.This {
		return false
	}
	if g.Len() != c.Len() {
		return false
	}

	for i := 0; i < g.Len(); i++ {
		if g.Out[i].Equal(c.Out[i]) == false {
			return false
		}
	}
	return true
}

// Add adds a subnode to the current node.
//
// An eventual nil root will not be bypassed.
func (g *Graph) Add(n interface{}) *Graph {
	if node, ok := n.(*Graph); ok {
		if node.IsNil() {
			for _, node2 := range node.Out {
				g.Out = append(g.Out, node2)
			}
		} else {
			g.Out = append(g.Out, node)
		}
		return node
	}

	gg := Graph{n, nil}
	g.Out = append(g.Out, &gg)
	return &gg
}

// AddNodes adds subnodes of the given Graph to the current node.
func (g *Graph) AddNodes(g2 *Graph) *Graph {

	if g2 != nil {
		for _, n := range g2.Out {
			g.Out = append(g.Out, n)
		}
	}
	return g
}

// addEqualNodes adds subnodes of the given Graph to the current node,
// if their content equals the given key. Optionally recurse into subnodes
// of the receiver graph.
func (g *Graph) addEqualNodes(g2 *Graph, key string, recurse bool) *Graph {

	if g2 != nil {
		for _, n := range g2.Out {
			if key == n.String() {
				g.AddNodes(n)
			}
			if recurse {
				g.addEqualNodes(n, key, true)
			}
		}
	}
	return g
}

// Copy adds a copy of the graph given to the current graph.
//
// Warning (from the Go faq): Copying an interface value makes a copy of the
// thing stored in the interface value. If the interface value holds a struct,
// copying the interface value makes a copy of the struct. If the interface
// value holds a pointer, copying the interface value makes a copy of the
// pointer, but not the data it points to.
func (g *Graph) Copy(c *Graph) {
	for _, n := range c.Out {
		nn := g.Add(n.This)
		nn.Copy(n)
	}
}

// Node returns the first subnode whose string value is equal to the given string.
// It returns nil if not found.
func (g *Graph) Node(s string) *Graph {

	for _, node := range g.Out {
		if s == node.String() {
			return node
		}
	}

	return nil
}

// GetAt returns a subnode by index, or nil if the index is out of range.
func (g *Graph) GetAt(i int) *Graph {
	if i >= len(g.Out) || i < 0 {
		return nil
	}

	return g.Out[i]
}

// Get(path) recurses a Graph following the given path and returns
// the result.
//
// This function returns consequently a *Graph. It may be a pointer within
// the recursed Graph (the receiver), or a newly created one. We leave the
// handling of specific types to the functions defined in get_types.go.
//
// OGDL Path:
// elements are separated by '.' or [] or {}
// index := [N]
// selector := {N}
// tokens can be quoted
//
// Future:
// .*., .**.
// ./regex/.
//
// Nil receiver behavior: return nil.
func (g *Graph) Get(s string) *Graph {
	if g == nil {
		return nil
	}
	// Parse the input string into a Path graph.
	path := NewPath(s)

	if path == nil {
		return nil
	}
	return g.get(path)
}

func (g *Graph) get(path *Graph) *Graph {
	if g == nil {
		return nil
	}

	iknow := true

	node := g

	// nodePrev = Upper level of current node, used in {}
	var nodePrev *Graph
	// elemPrev = previous path element, used in {}
	var elemPrev string

	// normalize the context graph that it allways has a nil root
	if !node.IsNil() {
		g = NilGraph()
		g.Add(node)
		node = g
	}

	for _, elem := range path.Out {

		c := elem.String()[0]

		if c == '!' {
			iknow = false
			c = elem.String()[1]

			switch c {

			case 'i':
				if elem.Len() == 0 {
					return nil
				}

				i, err := strconv.Atoi(elem.Out[0].String())
				if err != nil {
					return nil
				}
				nodePrev = node
				node = node.GetAt(i)
				if node == nil {
					return nil
				}
				elemPrev = node.String()

			case 's':
				if nodePrev == nil || nodePrev.Len() == 0 || len(elemPrev) == 0 {
					return nil
				}

				r := NilGraph()

				if elem.Len() == 0 {
					// This case is {}, meaning that we must return
					// all ocurrences of the token just before (elemPrev).
					// And that means creating a new Graph object.

					r.addEqualNodes(nodePrev, elemPrev, false)

					if r.Len() == 0 {
						return nil
					}
					node = r
				} else {
					i, err := strconv.Atoi(elem.Out[0].String())
					if err != nil || i < 0 {
						return nil
					}

					// {0} must still be handled: add it to r

					i++
					// of all the nodes with name elemPrev, select the ith.
					for _, nn := range nodePrev.Out {
						if nn.String() == elemPrev {
							i--
							if i == 0 {
								r.AddNodes(nn)
								node = r
								break
							}
						}
					}
					if i > 0 {
						return nil
					}
				}

			default:
				return nil
			}
		} else {
			iknow = true
			nodePrev = node
			elemPrev = elem.String()
			node = node.Node(elemPrev)
		}

		if node == nil {
			break
		}
	}

	if node == nil {
		return nil
	}

	// iknow true if the path includes the token that is now at the root of node.
	// We don't want to return what we already know.

	if iknow {
		if node.Len() == 1 {
			node = node.Out[0]
		} else {
			node2 := NilGraph()
			node2.Out = node.Out
			return node2
		}
	}

	// A nil node with one subnode makes no sense. Nil root nodes
	// are used as list containers.
	if node.IsNil() && node.Len() == 1 {
		return node.Out[0]
	}

	return node
}

// Delete removes all subnodes with the given value or content
func (g *Graph) Delete(n interface{}) {
	for i := 0; i < g.Len(); i++ {
		if g.Out[i].This == n {
			g.Out = append(g.Out[:i], g.Out[i+1:]...)
			i--
		}
	}
}

// DeleteN removes a subnode by index
func (g *Graph) DeleteAt(i int) {
	if i < 0 {
		return
	}
	if i >= g.Len() {
		return
	}
	g.Out = append(g.Out[:i], g.Out[i+1:]...)
}

// Set sets the first occurrence of the given path to the value given.
//
// TODO: Support indexes
func (g *Graph) Set(s string, val interface{}) *Graph {
	if g == nil {
		return nil
	}

	// Parse the input string into a Path graph.
	path := NewPath(s)

	if path == nil {
		return nil
	}
	return g.set(path, val)
}

func (g *Graph) set(path *Graph, val interface{}) *Graph {

	node := g

	i := 0
	var prev *Graph

	for ; i < len(path.Out); i++ {

		elem := path.Out[i]

		prev = node
		node = node.Node(elem.String())

		if node == nil {
			break
		}
	}

	if node == nil {
		node = prev

		for ; i < len(path.Out); i++ {
			elem := path.Out[i]
			node = node.Add(elem.This)
		}
	}

	node.Out = nil

	return node.Add(val)
}

// Text is the OGDL text emitter. It converts a Graph into OGDL text.
//
// Strings are quoted if they contain spaces, newlines or special
// characters. Null elements are not printed, and act as transparent nodes.
//
// BUG():Handle comments correctly.
//
func (g *Graph) Text() string {
	if g == nil {
		return ""
	}

	buffer := &bytes.Buffer{}

	g._text(0, buffer)

	// remove trailing \n

	s := buffer.String()

	if len(s) == 0 {
		return ""
	}

	if s[len(s)-1] == '\n' {
		s = s[0 : len(s)-1]
	}

	// unquote

	if s[0] == '"' {
		s = s[1 : len(s)-1]
		// But then also replace \"
		s = strings.Replace(s, "\\\"", "\"", -1)
	}

	return s
}

// _text is the private, lower level, implementation of Text().
// It takes two parameters, the level and a buffer to which the
// result is printed.
func (g *Graph) _text(n int, buffer *bytes.Buffer) {

	sp := ""
	for i := 0; i < n; i++ {
		sp += "  "
	}

	/*
	   When printing strings with newlines, there are two possibilities:
	   block or quoted. Block is cleaner, but limited to leaf nodes. If the node
	   is not leaf (it has subnodes), then we are forced to print a multiline
	   quoted string.

	   If the string has no newlines but spaces or special characters, then the
	   same rule applies: quote those nodes that are non-leaf, print a block
	   otherways.

	   [!] Cannot print blocks at level 0? Or can we?
	*/

	if strings.IndexAny(g.String(), "\n\r \t'\",()") != -1 {

		// print quoted, but not at level 0
		// Do not convert " to \" below if level==0 !
		if n > 0 {
			buffer.WriteString(sp[:len(sp)-1])
			buffer.WriteByte('"')
		}

		var c, cp byte

		cp = 0

		for i := 0; i < len(g.String()); i++ {
			c = g.String()[i] // byte, not rune
			if c == 13 {
				continue // ignore CR's
			} else if c == 10 {
				buffer.WriteByte('\n')
				buffer.WriteString(sp)
			} else if c == '"' && n>0 {
				if cp != '\\' {
					buffer.WriteString("\\\"")
				}
			} else {
				buffer.WriteByte(c)
			}
			cp = c
		}

		if n > 0 {
			buffer.WriteString("\"")
		}
		buffer.WriteString("\n")
	} else {
		if len(g.String()) == 0 {
			n--
		} else {
			buffer.WriteString(sp)
			buffer.WriteString(g.String())
			buffer.WriteByte('\n')
		}
	}

	for i := 0; i < len(g.Out); i++ {
		node := g.Out[i]
		node._text(n+1, buffer)
	}
}

// Substitute traverses the graph substituting all nodes with content
// equal to s by v.
func (g *Graph) Substitute(s string, v interface{}) {
	for _, n := range g.Out {
		if n.String() == s {
			n.This = v
		}
		n.Substitute(s, v)
	}

}
