// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bytes"
	"errors"
	"io"
)

type Parser struct {
	Lexer                     // Buffered byte and rune readed
	ev    *SimpleEventHandler // The output (event) stream
}

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

func NewBytesParser(buf []byte) *Parser {
	return NewParser(bytes.NewBuffer(buf))
}

func (p *Parser) Graph() *Graph {
	return p.ev.Tree()
}

// Ogdl is the main function for parsing OGDL text.
//
// An OGDL stream is a sequence of lines (a block
// of text or a quoted string can span multiple lines
// but is still parsed by Line())
//
//     Graph ::= Line* End
func (p *Parser) Ogdl() {
	n, u := p.Space()
	if u == 0 {
		return // Error
	}
	for i := n; i > 0; i-- {
		p.UnreadByte()
	}
	p.tree(n)
}

// tree reads all lines with indentation >= ns
func (p *Parser) tree(ns int) {
	for {
		b, err := p.line(ns)
		if !b || err != nil {
			break
		}
	}
	p.End()
}

// line processes an OGDL line or a multiline scalar.
//
// - A Line is composed of scalars and groups.
// - A Scalar is a Quoted or a String.
// - A Group is a sequence of Scalars enclosed in parenthesis
// - Scalars can be separated by commas or by space
// - The last element of a line can be a Comment, or a Block
//
// The indentation of the line and the Scalar sequences and Groups on it define
// the tree structure characteristic of OGDL level 1.
//
//    Line ::= Space(n) Sequence? ((Comment? Break)|Block)?
//
// Anything other than one Scalar before a Block should be an syntax error.
// Anything after a closing ')' that is not a comment is a syntax error, thus
// only one Group per line is allowed. That is because it would be difficult to
// define the origin of the edges pointing to what comes after a Group.
//
// Indentation rules:
//
//   a           -> level 0
//     b         -> level 1
//     c         -> level 1
//       d       -> level 2
//      e        -> level 2
//    f          -> level 1
//
func (p *Parser) line(ns int) (bool, error) {

	n, u := p.Space()

	if n < ns {
		return false, nil
	}

	if n > ns {
		for i := n; i > 0; i-- {
			p.UnreadByte()
		}
		p.ev.Inc()
		p.tree(n)
		p.ev.Dec()
		return true, nil
	}

	level := p.ev.Level()

	if u == 0 && n == 0 {
		return false, errors.New("non-uniform space")
	}

	if p.End() {
		return false, nil
	}

	// We should not have a Comma here, but lets ignore it.
	if p.PeekByte() == ',' {
		p.Byte()
		p.Space() // Eat eventual space characters
	}

	for {

		// Can be:

		// Scalar
		// Break
		// End
		// Comment
		// Block

		if p.End() {
			return false, nil
		}

		if p.Break() {
			break
		}

		if p.Comment() {
			break
		}

		s, ok := p.Block(n)

		if ok {
			p.ev.Add(s)
			p.Break()
			break
		}

		b, ok := p.Scalar(n)

		if ok {
			p.ev.Add(b)
		}

		if p.Break() {
			break
		}

		p.Space()

		if p.PeekByte() == ',' {
			p.Byte()
			p.Space()
			// After a comma, reset the level to that of the start of this Line.
			p.ev.SetLevel(n)
		} else {
			p.ev.Inc()
		}
	}

	p.ev.SetLevel(level)

	return true, nil

}
