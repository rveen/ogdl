// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"strings"
)

// TODO:
// - check if types can be entered accidentaly in text (!e, etc)
// - desligar parser de textual
// - error (not panic)

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

/*
Parser is used to parse textual OGDL streams into Graph objects.

There are several types of functions here:

   - Read and Unread: elementary character handling.
   - Character classifiers.
   - Elementary productions that return a bool.
   - Productions that return a string (or nil).
   - Productions that produce an event.

The OGDL parser doesn't need to know about Unicode. The character
classification relies on values < 127, thus in the ASCII range,
which is also part of Unicode.

Note: On the other hand it would be stupid not to recognize for example
Unicode quotation marks if we know that we have UTF-8. But when do we
know for sure?

BUG(): Level 2 (graphs) not implemented
*/
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

	// the number of characters after a NL was found
	// (Used in Quoted)
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
	p.pOgdl()
	return p.Graph()
}

func ParseFile(s string) *Graph {
	p := NewFileParser(s)
	p.pOgdl()
	return p.Graph()
}

func (p *Parser) Graph() *Graph {
	return p.ev.Graph()
}

func (p *Parser) GraphTop(s string) *Graph {
	return p.ev.GraphTop(s)
}

/* pOgdl is the main function for parsing OGDL text.

   An OGDL stream is a sequence of lines (a block
   of text or a quoted string can span multiple lines
   but is still a single node)

     Graph ::= Line* End
*/
func (p *Parser) pOgdl() error {

	for {
		more, err := p.pLine()
		if err != nil {
			return err
		}
		if !more {
			break
		}
	}
	p.End()

	return nil
}

/* Line processes an OGDL line or a multiline scalar.

 - A Line is composed of scalars and groups.
 - A Scalar is a Quoted or a String.
 - A Group is a sequence of Scalars enclosed in parenthesis
 - Scalars can be separated by commas or by space
 - The last element of a line can be a Comment, or a Block

The indentation of the line and the Scalar sequences and Groups on it define
the tree structure characteristic of OGDL level 1.

    Line ::= Space(n) Sequence? ((Comment? Break)|Block)?

Anything other than one Scalar before a Block should be an syntax error.
Anything after a closing ')' that is not a comment is a syntax error, thus
only one Group per line is allowed. That is because it would be difficult to
define the origin of the edges pointing to what comes after a Group.

Indentation rules:

   a           -> level 0
     b         -> level 1
     c         -> level 1
       d       -> level 2
      e        -> level 2
    f          -> level 1

*/
func (p *Parser) pLine() (bool, error) {

	sp, n := p.Space()

	// if a line begins with non-uniform space, throw a syntax error.
	if sp && n == 0 {
		errors.New("OGDL syntax error: non-uniform space")
	}

	if p.End() {
		return false, nil
	}

	// We should not have a Comma here, but lets ignore it.
	if p.NextByteIs(',') {
		p.Space() // Eat eventual space characters
	}

	/* indentation TO level

	   The number of spaces (indentation) for each level is stored in
	   p.ind[level]
	*/

	l := 0

	if n != 0 {
		l = 1
		for {
			if p.ind[l] == 0 {
				break
			}
			if p.ind[l] >= n {
				break
			}
			l++
		}
	}

	p.ind[l] = n
	p.ev.SetLevel(l)

	// Now we can expect a sequence of scalars, groups, and finally
	// a block or comment.

	for {

		gr, err := p.Group()

		if gr {

		} else if err != nil {
			return false, err
		} else if p.Comment() {
			p.Space()
			p.Break()
			break
		} else {
			s := p.Block()

			if len(s) > 0 {
				p.ev.Add(s)
				p.Break()
				break
			} else {
				b := p.Scalar()
				if b != nil {
					p.ev.AddBytes(b)
				} else {
					p.Break()
					break
				}
			}
		}

		p.Space()

		co := p.NextByteIs(',')

		if co {
			p.Space()
			p.ev.SetLevel(l)
		} else {
			p.ev.Inc()
		}

	}

	// Restore the level to that at the beginning of the line.
	p.ev.SetLevel(l)

	return true, nil
}

// nextByteIs tests if the next character in the
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

/* ---------------------------------------------
   Elementary byte handling

   OGDL doesn't need to look ahead further than 2 chars.
   --------------------------------------------- */

// read reads the next byte out of the stream.
//
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
// XXX TODO: line-- if newline
func (p *Parser) Unread() {
	p.lastn++
	p.lastnl--
}
