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

// Nodes containing these strings are special
const (
	TypeExpression = "!e"
	TypePath       = "!p"
	TypeVariable   = "!v"
	TypeSelector   = "!s"
	TypeIndex      = "!i"
	TypeGroup      = "!g"
	TypeTemplate   = "!t"

	TypeIf    = "!if"
	TypeEnd   = "!end"
	TypeElse  = "!else"
	TypeFor   = "!for"
	TypeBreak = "!break"
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

// NewStringParser creates an OGDL parser from a string 
func NewStringParser(s string) *Parser {
	return &Parser{strings.NewReader(s), NewEventHandler(), make([]int, 32), [2]int{0, 0}, 0, 0, 1, 0}
}

// NewParser creates an OGDL parser from a generic io.Reader
func NewParser(r io.Reader) *Parser {
	return &Parser{bufio.NewReader(r), NewEventHandler(), make([]int, 32), [2]int{0, 0}, 0, 0, 1, 0}
}

// NewFileParser creates an OGDL parser that reads from a file
func NewFileParser(s string) *Parser {
	b, err := ioutil.ReadFile(s)
	if err != nil || len(b) == 0 {
		return nil
	}

	buf := bytes.NewBuffer(b)
	return &Parser{buf, NewEventHandler(), make([]int, 32), [2]int{0, 0}, 0, 0, 1, 0}
}

// NewBytesParser creates an OGDL parser from a []byte source 
func NewBytesParser(b []byte) *Parser {
	buf := bytes.NewBuffer(b)
	return &Parser{buf, NewEventHandler(), make([]int, 32), [2]int{0, 0}, 0, 0, 1, 0}
}

// Parse parses OGDL text contained in a byte array. It returns a *Graph 
func Parse(b []byte) *Graph {
	p := NewBytesParser(b)
	p.Ogdl()
	return p.Graph()
}

// ParseString parses OGDL text from the given string. It returns a *Graph
func ParseString(s string) *Graph {
	p := NewBytesParser([]byte(s))
	p.Ogdl()
	return p.Graph()
}

// ParseFile parses OGDL text contained in a file. It returns a Graph
func ParseFile(s string) *Graph {
	p := NewFileParser(s)
	if p==nil {
	    return nil
	}
	p.Ogdl()
	return p.Graph()
}

// Graph returns the *Graph object associated with this parser (where root
// where the OGDL tree is build on).
func (p *Parser) Graph() *Graph {
	return p.ev.Graph()
}

// GraphTop returns the *Graph object associated with this parser (where root
// where the OGDL tree is build on). Additionally, the name of the root node
// is set to the given string.
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

// setLevel sets the nesting level for a given indentation (number of spaces)
// This function is used by the line() production for parsing OGDL text.
//
// setLevel sets ind[lev] = n, all ind[>lev] = 0 and assures that 
// ind[0..lev-1] has increasing n, adjusting n if necessary.
func (p *Parser) setLevel(lev, n int) {

	// Set ind[level] to the number of spaces + 1 (zero is nil)
	p.ind[lev] = n + 1

	// Fill holes
    for i:=1; i<lev; i++ {
        if p.ind[i] < p.ind[i-1] {
            p.ind[i] = p.ind[i-1]
        }
    }
    
    for i:=lev+1; i<len(p.ind); i++ {
        p.ind[i]=0
    }
}

// getLevel returns the nesting level corresponding to the given indentation.
// This function is used by the line() production for parsing OGDL text.
// 
// getLevel returns the level for which ind[level] is equal or higher than n.
func (p *Parser) getLevel(n int) int {

    l := 0
    
	for i := 0; i < len(p.ind); i++ {
		if p.ind[i] >= n {
			return i
		}
		if i!=0 && p.ind[i] == 0 {
			l = i - 1
			break
		}
	}
	if l<0 {
	    return 0
	}
	return l
}

/* 
  The following functions are public in order for the Parser to be used
  outside of the current package
*/

// Emit sends a string to the event handler
func (p *Parser) Emit(s string) {
    p.ev.Add(s)
}

// EmitBytes sends a byte array to the event handler
func (p *Parser) EmitBytes(b []byte) {
    p.ev.AddBytes(b)
}

// Inc increments the level by 1
func (p *Parser) Inc() {
    p.ev.Inc()
}

// Dec decrements the level by 1
func (p *Parser) Dec() {
    p.ev.Dec()
}