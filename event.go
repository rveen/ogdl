// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

// EventHandler receives events and produces a Graph.
type EventHandler struct {
	level int
	gl    []*Graph
}

// NewEventHandler creates an event handler that produces a Graph object
// from the events received.
func NewEventHandler() EventHandler {
	return EventHandler{}
}

// AddBytes creates a node at the current level, with the given byte array as content.
func (e *EventHandler) AddBytes(b []byte) bool {

	if len(e.gl) == 0 {
		e.gl = append(e.gl, NilGraph())
	}

	for len(e.gl) < e.level+2 {
		e.gl = append(e.gl, nil)
	}

	if e.gl[e.level] == nil {
		return false
	}

	e.gl[e.level+1] = e.gl[e.level].Add(b)
	return true
}

// Add creates a node at the current level.
//
// Only one error is possible: an empty graph where we should be writing the
// event. It that case, false is returned.
func (e *EventHandler) Add(s string) bool {

	// Create a transparent node to start with,
	// or else events at level 0 will overwrite
	// each other.
	if len(e.gl) == 0 {
		e.gl = append(e.gl, NilGraph())
	}

	for len(e.gl) < e.level+2 {
		e.gl = append(e.gl, nil)
	}

	// Protection against holes can also be
	// done at other places in this package.
	if e.gl[e.level] == nil {
		return false
	}

	e.gl[e.level+1] = e.gl[e.level].Add(s)
	return true
}

// Delete removes the last event added
func (e *EventHandler) Delete() {
	g := e.gl[e.level]
	n := g.Len()
	g.DeleteAt(n - 1)

	e.gl[e.level+1] = g.Out[n-2]
}

// AddAt creates a node at the specified level
func (e *EventHandler) AddAt(s string, l int) {
	e.level = l - 1
	e.Add(s)
}

// AddBytesAt creates a node at the specified level, with the byte slice
// as content.
func (e *EventHandler) AddBytesAt(b []byte, l int) {
	e.level = l - 1
	e.AddBytes(b)
}

// Level returns the current level
func (e *EventHandler) Level() int {
	return e.level
}

// SetLevel sets the current level
func (e *EventHandler) SetLevel(l int) {
	e.level = l
}

// Inc increments the current level by 1.
func (e *EventHandler) Inc() {
	e.level++
}

// Dec decrements the current level by 1.
func (e *EventHandler) Dec() {
	if e.level > 0 {
		e.level--
	}
}

// Graph returns the Graph object built from
// the events sent to this event handler.
//
func (e *EventHandler) Graph() *Graph {

	// It could happen that Graph() is requested
	// while no event has been sent, and thus
	// e.gl hasn't been initialized yet.
	if len(e.gl) == 0 {
		return nil
	}

	// Check if e.gl[0] is transparent and has only one subnode.
	g := e.gl[0]
	if g == nil {
		return nil
	}
	if g.Len() == 1 && g.IsNil() {
		return g.Out[0]
	}
	return g
}

// GraphTop returns the Graph object built from
// the events sent to this event handler, and sets
// the root node to the string given.
func (e *EventHandler) GraphTop(s string) *Graph {

	if len(e.gl) == 0 {
		return nil
	}

	g := e.gl[0]
	if g == nil {
		return nil
	}
	if g.IsNil() && len(s) > 0 {
		g.This = s
		return g
	}
	return g
}
