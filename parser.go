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
