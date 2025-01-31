// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bytes"
	"reflect"
	"strings"
)

// Graph is a node with outgoing pointers to other Graph objects.
// It is implemented as a named list.
type Graph struct {
	This interface{}
	Out  []*Graph
}

// New returns a pointer to Graph initialized to the object given.
func New(n interface{}) *Graph {
	return &Graph{n, nil}
}

// Len returns the number of subnodes (outgoing edges, out degree) of this node.
func (g *Graph) Len() int {
	if g == nil {
		return -1
	}
	return len(g.Out)
}

// ThisType returns the name of the native type contained in the current node.
func (g *Graph) ThisType() string {
	return reflect.TypeOf(g.This).String()
}

// Equals returns true if the given graph and the receiver graph are equal.
func (g *Graph) Equals(c *Graph) bool {

	if c.This != g.This {
		return false
	}
	if g.Len() != c.Len() {
		return false
	}

	for i := 0; i < g.Len(); i++ {
		if !g.Out[i].Equals(c.Out[i]) {
			return false
		}
	}
	return true
}

// Add adds a subnode to the current node.
func (g *Graph) Add(n interface{}) *Graph {

	if g == nil || n == nil {
		return nil
	}

	if node, ok := n.(*Graph); ok && node != nil {
		g.Out = append(g.Out, node)
		return node
	}

	gg := Graph{n, nil}
	g.Out = append(g.Out, &gg)
	return &gg
}

// Add adds a subnode to the current node.
func (g *Graph) addNodes(n interface{}) *Graph {

	if g == nil || n == nil {
		return nil
	}

	if node, ok := n.(*Graph); ok && node != nil {
		g.Out = append(g.Out, node.Out...)
		return node
	}

	gg := Graph{n, nil}
	g.Out = append(g.Out, &gg)
	return &gg
}

// AddNodes adds subnodes of the given Graph to the current node.
func (g *Graph) AddNodes(g2 *Graph) *Graph {

	if g == nil || g2 == nil {
		return nil
	}

	if g2 != nil {
		g.Out = append(g.Out, g2.Out...)
	}
	return g
}

// addEqualNodes adds subnodes of the given Graph to the current node,
// if their content equals the given key. Optionally recurse into subnodes
// of the receiver graph.
func (g *Graph) addEqualNodes(g2 *Graph, key string, recurse bool) *Graph {
	if g2 != nil {
		for _, n := range g2.Out {
			if key == _string(n.This) {
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
	if g == nil || c == nil {
		return
	}
	for _, n := range c.Out {
		nn := g.Add(n.This)
		nn.Copy(n)
	}
}

// Clone returns a copy of the current graph.
//
// Warning (from the Go faq): Copying an interface value makes a copy of the
// thing stored in the interface value. If the interface value holds a struct,
// copying the interface value makes a copy of the struct. If the interface
// value holds a pointer, copying the interface value makes a copy of the
// pointer, but not the data it points to.
func (g *Graph) Clone() *Graph {
	if g == nil {
		return nil
	}

	c := New(nil)
	c.This = g.This

	for _, n := range g.Out {
		c.Out = append(c.Out, n.Clone())
	}
	return c
}

// Node returns the first subnode whose string value is equal to the given string.
// It returns nil if not found.
func (g *Graph) Node(s string) *Graph {
	if g == nil || g.Out == nil {
		return nil
	}
	for _, node := range g.Out {
		if s == _string(node.This) {
			return node
		}
	}

	return nil
}

// NodeOrNew returns the first subnode whose string value is equal to the given string.
// It creates the node if not found.
func (g *Graph) NodeOrNew(s string) *Graph {
	if g == nil || g.Out == nil {
		return nil
	}
	for _, node := range g.Out {
		if s == _string(node.This) {
			return node
		}
	}

	return g.Add(s)
}

// Create returns the first subnode whose string value is equal to the given string,
// with its subnodes deleted. If not found, the node is created and returned.
func (g *Graph) Create(s string) *Graph {
	n := g.Node(s)
	if n == nil {
		return g.Add(s)
	}
	n.Clear()
	return n
}

// GetAt returns a subnode by index, or nil if the index is out of range.
func (g *Graph) GetAt(i int) *Graph {
	if i >= len(g.Out) || i < 0 {
		return nil
	}

	return g.Out[i]
}

// Get recurses a Graph following a given path and returns the result.
//
// This function returns a *Graph in any condition. When there is nothing to
// return, a nil Graph is returned. This behavior is designed so that
// the next function in a chain never gets an invalid receiver, avoiding null
// pointer errors.
//
// OGDL Path:
// elements are separated by '.' or [] or {}
// index := [N]
// selector := {N}
// tokens can be quoted
func (g *Graph) Get(s string) *Graph {
	if g == nil {
		return nil
	}
	path := NewPath(s)
	g, _ = g.getPath(path)
	return g
}

// Delete removes all subnodes with the given content
func (g *Graph) Delete(n interface{}) {

	if g == nil {
		return
	}
	for i := 0; i < g.Len(); i++ {
		if g.Out[i].This == n {
			if i < (g.Len() - 1) {
				g.Out = append(g.Out[:i], g.Out[i+1:]...)
			} else {
				g.Out = g.Out[:i]
			}
			i--
		}
	}
}

// Clear removes all subnodes
func (g *Graph) Clear() {

	if g == nil || g.Out == nil {
		return
	}
	g.Out = nil
}

// DeleteAt removes a subnode by index
func (g *Graph) DeleteAt(i int) {
	if i < 0 || i >= g.Len() {
		return
	}
	if i < (g.Len() - 1) {
		g.Out = append(g.Out[:i], g.Out[i+1:]...)
	} else {
		g.Out = g.Out[:i]
	}
}

// Set sets the first occurrence of the given path to the value given.
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

// TODO: Clean this code:
func (g *Graph) set(path *Graph, val interface{}) *Graph {

	node := g

	i := 0
	var prev *Graph

	for ; i < len(path.Out); i++ {

		prev = node

		elem := path.Out[i]
		if elem.ThisString() == TypeIndex {
			i := elem.Int64()
			if len(node.Out) <= int(i) {
				o := make([]*Graph, i+1)
				for j, n := range node.Out {
					o[j] = n
				}
				node.Out = o
			}
			node.Out[i] = New(val)
			return node.Out[i]
		}
		node = node.Node(elem.ThisString())

		if node == nil {
			break
		}
	}

	if node == nil {
		node = prev

		for ; i < len(path.Out); i++ {
			elem := path.Out[i]

			if elem.ThisString() == TypeIndex {
				i := elem.Int64()
				if len(node.Out) <= int(i) {
					o := make([]*Graph, i+1)
					for j, n := range node.Out {
						o[j] = n
					}
					node.Out = o
				}
				node.Out[i] = New(val)
				return node.Out[i]
			}

			node = node.Add(elem.This)
		}
	}

	node.Out = nil

	return node.addNodes(val)
}

// Text is the OGDL text emitter. It converts a Graph into OGDL text.
//
// Strings are quoted if they contain spaces, newlines or special
// characters. Null elements are not printed, and act as transparent nodes.
func (g *Graph) Text() string {
	if g == nil {
		return ""
	}

	buffer := &bytes.Buffer{}

	// Do not print the 'root' node
	for _, node := range g.Out {
		node._text(0, buffer, false)
	}

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

// Show prints the Graph as text including this (the top) node.
func (g *Graph) Show() string {
	if g == nil {
		return ""
	}

	buffer := &bytes.Buffer{}

	g._text(0, buffer, true)

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
func (g *Graph) _text(n int, buffer *bytes.Buffer, show bool) {

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

	s := "_"
	if g != nil {
		s = _string(g.This)
	}

	if strings.ContainsAny(s, "\n\r \t'\",()") {

		// print quoted, but not at level 0
		// Do not convert " to \" below if level==0 !
		if n > 0 {
			buffer.WriteString(sp) /* [:len(sp)-1]) */
			buffer.WriteByte('"')
		}

		var c, cp byte

		cp = 0

		for i := 0; i < len(s); i++ {
			c = s[i] // byte, not rune
			if c == 13 {
				continue // ignore CR's
			} else if c == 10 {
				buffer.WriteByte('\n')
				buffer.WriteString(sp)
			} else if c == '"' && n > 0 {
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
		if len(s) == 0 && !show {
			n--
		} else {
			if len(s) == 0 && show {
				s = "_"
			}
			buffer.WriteString(sp)
			buffer.WriteString(s)
			buffer.WriteByte('\n')
		}
	}

	if g != nil {
		for i := 0; i < len(g.Out); i++ {
			node := g.Out[i]
			node._text(n+1, buffer, show)
		}
	}
}

// Substitute traverses the graph substituting all nodes with content
// equal to s by v.
func (g *Graph) Substitute(s string, v interface{}) {
	if g == nil || g.Out == nil {
		return
	}
	for _, n := range g.Out {
		if _string(n.This) == s {
			n.This = v
		}
		n.Substitute(s, v)
	}

}

// Merge the given graph into the current (receiver) graph
func (g *Graph) Merge(src *Graph) {

	for _, node := range src.Out {
		n := g.Node(node.String())

		if n == nil {
			g.Add(node)
		} else {
			n.Out = nil
			n.Merge(node)
		}
	}
}
