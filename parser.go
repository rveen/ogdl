// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bytes"
	"io"
	"io/ioutil"
)

// Parser embeds Lexer and holds some state
type Parser struct {
	Lexer                     // Buffered byte and rune readed
	ev    *SimpleEventHandler // The output (event) stream
}

// NewParser return a new Parser from a Reader
func NewParser(rd io.Reader) *Parser {
	p := Parser{}
	p.rd = rd
	p.lastByte = bufSize
	p.buf = make([]byte, bufSize)
	p.ev = &SimpleEventHandler{}
	p.r = -1
	p.fill()
	return &p
}

// NewBytesParser returns a new Parser from a byte array
func NewBytesParser(buf []byte) *Parser {
	return NewParser(bytes.NewBuffer(buf))
}

// NewStringParser returns a new Parser from a string
func NewStringParser(s string) *Parser {
	return NewParser(bytes.NewBuffer([]byte(s)))
}

// Graph returns the parser tree
func (p *Parser) Graph() *Graph {
	return p.ev.Tree()
}

// Handler returns the event handler being used
func (p *Parser) Handler() *SimpleEventHandler {
	return p.ev
}

// FromBytes parses OGDL text contained in a byte array. It returns a *Graph
func FromBytes(b []byte) *Graph {
	p := NewParser(bytes.NewBuffer(b))
	p.Ogdl()
	return p.Graph()
}

// FromString parses OGDL text from the given string. It returns a *Graph
func FromString(s string) *Graph {
	p := NewParser(bytes.NewBuffer([]byte(s)))
	p.Ogdl()
	return p.Graph()
}

// FromStringTypes parses OGDL text from the given string. It returns a *Graph.
// Basic types found in the string are converted to their correspongind Go types
// (either string | int64 | float64 | bool).
func FromStringTypes(s string) *Graph {
	p := NewParser(bytes.NewBuffer([]byte(s)))
	p.OgdlTypes()
	return p.Graph()
}

// FromReader parses OGDL text coming from a generic io.Reader
func FromReader(r io.Reader) *Graph {
	p := NewParser(r)
	p.Ogdl()
	return p.Graph()
}

// FromFile parses OGDL text contained in a file. It returns a Graph
func FromFile(s string) *Graph {
	b, err := ioutil.ReadFile(s)
	if err != nil || len(b) == 0 {
		return nil
	}

	buf := bytes.NewBuffer(b)
	p := NewParser(buf)
	p.Ogdl()
	return p.Graph()
}

// Some usefull functions to extended the Parser and use it in other places

// Emit outputs a string event at the current level. This will show up in the graph
func (p *Parser) Emit(s string) {
	p.ev.Add(s)
}

// Inc increases the event handler level by one
func (p *Parser) Inc() {
	p.ev.Inc()
}

// Dec decreses the event handler level by one
func (p *Parser) Dec() {
	p.ev.Dec()
}
