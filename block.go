// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"errors"
)

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
	p.tree(n, false)
}

// OgdlTypes is the main function for parsing OGDL text.
//
// This version tries to convert unquoted strings that can be parsed as ints, floats
// or bools to their corresponding type in Go (string | int64 | float64 | bool).
func (p *Parser) OgdlTypes() {
	n, u := p.Space()
	if u == 0 {
		return // Error
	}
	for i := n; i > 0; i-- {
		p.UnreadByte()
	}
	p.tree(n, true)
}

// tree reads all lines with indentation >= ns
func (p *Parser) tree(ns int, types bool) {
	for {
		b, err := p.line(ns, types)
		if !b || err != nil {
			break
		}
	}
	p.End()
}

// line processes an OGDL line or a multiline scalar.
//
// - A Line is composed of one or more scalars separated by space
// - A Scalar is a Quoted or a String.
// - The last element of a line can be a Comment, or a Block
//
// The indentation of the line and the Scalar sequences define
// the tree structure characteristic of OGDL level 1.
//
//    Line ::= Space(n) Sequence? ((Comment? Break)|Block)?
//
// Anything other than one Scalar before a Block should be an syntax error.
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
func (p *Parser) line(ns int, types bool) (bool, error) {

	n, u := p.Space()

	if n < ns {
		for i := n; i > 0; i-- {
			p.UnreadByte()
		}
		return false, nil
	}

	if n > ns {
		for i := n; i > 0; i-- {
			p.UnreadByte()
		}
		p.ev.Inc()
		p.tree(n, types)
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

		if !types {
			b, ok := p.Scalar(n)

			if ok {
				p.ev.Add(b)
			}
		} else {
			b, ok := p.ScalarType(n)

			if ok {
				p.ev.AddItf(b)
			}
		}

		if p.Break() {
			break
		}

		p.Space()
		p.ev.Inc()
	}

	p.ev.SetLevel(level)

	return true, nil

}
