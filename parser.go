// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"strings"
)

const (
	TYPE_EXPRESSION = "!e"
	TYPE_PATH       = "!p"
	TYPE_VARIABLE   = "!v"
	TYPE_SELECTOR   = "!s"
	TYPE_INDEX      = "!i"
	TYPE_GROUP      = "!g"
	TYPE_TEMPLATE   = "!t"

	TYPE_IF    = "!if"
	TYPE_END   = "!end"
	TYPE_ELSE  = "!else"
	TYPE_FOR   = "!for"
	TYPE_BREAK = "!break"
)

// Parser is used to parse textual OGDL streams, paths, empressions and
// templates into Graph objects.
//
// Simple productions return a scalar (normally a string), more complex ones
// write to and event handler.
//
// BUG(): Level 2 (graphs) not implemented.
type Parser struct {
	// The input (byte) stream
	in io.ByteReader

	// The output (event) stream
	ev EventHandler

	// ind holds indentation at different levels, that is,
	// the number of spaces at each level.
	ind []int

	// last holds the 2 last characters read.
	// We need 2 characters of look-ahead (for Block()).
	last [2]int

	// unread index
	lastn int

	// the number of characters after a NL was found (used in Quoted)
	lastnl int

	// line keeps track of the line number
	line int

	// saved spaces at end of block
	spaces int
}

func NewStringParser(s string) *Parser {
	return &Parser{strings.NewReader(s), NewEventHandler(), make([]int, 32), [2]int{0, 0}, 0, 0, 1, 0}
}

func NewParser(r io.Reader) *Parser {
	return &Parser{bufio.NewReader(r), NewEventHandler(), make([]int, 32), [2]int{0, 0}, 0, 0, 1, 0}
}

func NewFileParser(s string) *Parser {
	b, err := ioutil.ReadFile(s)
	if err != nil || len(b) == 0 {
		return nil
	}

	buf := bytes.NewBuffer(b)
	return &Parser{buf, NewEventHandler(), make([]int, 32), [2]int{0, 0}, 0, 0, 1, 0}
}

func NewBytesParser(b []byte) *Parser {
	buf := bytes.NewBuffer(b)
	return &Parser{buf, NewEventHandler(), make([]int, 32), [2]int{0, 0}, 0, 0, 1, 0}
}

func Parse(b []byte) *Graph {
	p := NewBytesParser(b)
	p.Ogdl()
	return p.Graph()
}

func ParseString(s string) *Graph {
	p := NewBytesParser([]byte(s))
	p.Ogdl()
	return p.Graph()
}

func ParseFile(s string) *Graph {
	p := NewFileParser(s)
	p.Ogdl()
	return p.Graph()
}

func (p *Parser) Graph() *Graph {
	return p.ev.Graph()
}

func (p *Parser) GraphTop(s string) *Graph {
	return p.ev.GraphTop(s)
}

// NextByteIs tests if the next character in the
// stream is the one given as parameter, in which
// case it is consumed.
//
func (p *Parser) NextByteIs(c int) bool {
	ch := p.Read()
	if ch == c {
		return true
	}
	p.Unread()
	return false
}

// Read reads the next byte out of the stream.
func (p *Parser) Read() int {

	var c int

	if p.lastn > 0 {
		p.lastn--
		c = p.last[p.lastn]
	} else {
		i, _ := p.in.ReadByte()
		c = int(i)
		p.last[1] = p.last[0]
		p.last[0] = c
	}

	if c == 10 {
		p.lastnl = 0
		p.line++
	} else {
		p.lastnl++
	}

	return c
}

// Unread puts the last readed character back into the stream.
// Up to two consecutive Unread()'s can be issued.
//
// BUG: line-- if newline
func (p *Parser) Unread() {
	p.lastn++
	p.lastnl--
}
