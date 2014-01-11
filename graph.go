// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bytes"
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

// Len returns the number of subnodes (outgoing edges) of this node.
// = out degree
func (g *Graph) Len() int {
	if g == nil {
		return -1
	}
	return len(g.Out)
}

// Depth returns the depth of the graph if it is a tree, or -1 if it has
// cycles.
//
// XXX Check for cycles
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

// Add adds a subnode to the current node.
//
// An eventual nil root will not be added (it will be bypassed).
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

// Node returns the first subnode with the specified name.
// Attention: converts values to string first
func (g *Graph) Node(n interface{}) *Graph {

	for _, node := range g.Out {
		if _string(n) == node.String() {
			return node
		}
	}

	return nil
}

// GetAt returns a subnode by index.
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

	strip := true

	node := g

	// normalize the context graph that it allways has a nil root
	if !node.IsNil() {
		g = NilGraph()
		g.Add(node)
		node = g
	}

	for _, elem := range path.Out {

		c := elem.String()[0]

		if c == '!' {
			strip = false
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
				node = node.GetAt(i)
			case 's':
				if elem.Len() == 0 {
					// This case is {}, meaning that we must return
					// all ocurrences of the token just before.
					// And that means creating a new Graph object.
					// BUG(): TO-DO
					return nil
				}
				i, err := strconv.Atoi(elem.Out[0].String())
				if err != nil {
					return nil
				}
				i++
				for _, subnode := range node.Out {
					if elem.String() == subnode.String() {
						i--
					}
					if i == 0 {
						node = subnode
					}
				}

			case 'x':
				var ok bool
				node, ok = node.GetSimilar(elem.String())
				if !ok {
					return nil
				}
			default:
				return nil

			}
		} else {
			strip = true
			node = node.Node(elem.String())
		}

		if node == nil {
			break
		}
	}

	if strip && (node != nil) {
		if node.Len() == 1 && node.Out[0].Len() == 0 {
			return node.Out[0]
		}
		node2 := NilGraph()
		node2.Out = node.Out
		return node2
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
//
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
// Strings need to be quoted if they contain spaces, newlines or special
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

// print is the private, lower level, implementation of String.
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

		/* print quoted */
		buffer.WriteString(sp)
		buffer.WriteByte('"')

		var c byte

		for i := 0; i < len(g.String()); i++ {
			c = g.String()[i] // byte, not rune
			if c == 13 {
				continue // just ignore CR
			} else if c == 10 {
				buffer.WriteByte('\n')
				buffer.WriteString(sp)
				//buffer.WriteByte(' ')
			} else if c == '"' {
				buffer.WriteString("\\\"") // BUG(): check if \ was already there
			} else {
				buffer.WriteByte(c)
			}
		}

		buffer.WriteString("\"\n")
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

func (g *Graph) Substitute(s string, v interface{}) {
	for _, n := range g.Out {
		if n.String() == s {
			n.This = v
		}
		n.Substitute(s, v)
	}

}
