// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

// Path is used to parse an OGDL path, given as an Unicode string, into a Graph object.
//
// Syntax:
//
// path ::= token ('.' element)*
//
// element ::= token | integer | quoted | group | index | selector
//
// (Dot optional before Group, Index, Selector)
//
// group := '(' Expression [[,] Expression]* ')'
// index := '[' Expression ']'
// selector := '{' Expression '}'
//
// Example:
//
//   a.b(c, d+1).e[1+r].b{5}
func NewPath(s string) *Graph {
	parse := NewStringParser(s)
	parse.Path()
	return parse.GraphTop(TYPE_PATH)
}

// A path is a variable in path format,
// and must begin with a letter.
func (p *Parser) Path() bool {

	c := p.Read()
	p.Unread()

	if !IsLetter(c) {
		return false
	}

	var b []byte
	var begin = true
	var anything = false
	var ok bool
	var err error

	for {

		// Expect: token | quoted | index | group | selector | dot,
		// or else we abort.

		// A dot is requiered before a token or quoted, except at
		// the beginning

		if !p.NextByteIs('.') && !begin {
			// If not [, {, (, break

			c = p.Read()
			p.Unread()

			if c != '[' && c != '(' && c != '{' {
				break
			}
		}

		begin = false

		b = p.Quoted()
		if b != nil {
			p.ev.AddBytes(b)
			anything = true
			continue
		}

		b = p.Number()
		if b != nil {
			p.ev.AddBytes(b)
			anything = true
			continue
		}

		b = p.Token()
		if b != nil {
			p.ev.AddBytes(b)
			anything = true
			continue
		}

		if p.Index() {
			anything = true
			continue
		}

		if p.Selector() {
			anything = true
			continue
		}

		ok, err = p.Args()
		if ok {
			anything = true
			continue
		} else {
			if err != nil {
				return false // XXX
			}
		}

		break
	}

	return anything
}
