// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bytes"
	"errors"
)

/* sequence ::= (Scalar|Group) (Space? (Comma? Space?) (Scalar|Group))*

   [!] with the requirement that after a group a comma is required if there are more elements.

   Examples:
     a b c
     a b,c
     a(b,c)
     (a b,c)
     (b,c),(d,e) <-- This can be handled
     a (b c) d   <-- This is an error


   This method returns two booleans: if there has been a sequence, and if the last element was a Group
*/

func (p *Parser) Sequence() (bool, bool, error) {

	i := p.ev.Level()

	wasGroup := false
	n := 0

	for {
		gr, err := p.Group()
		if gr {
			wasGroup = true
		} else if err != nil {
			return false, false, err
		} else {
			b := p.Scalar()

			if b == nil {
				return n > 0, wasGroup, nil
			}
			wasGroup = false
			p.ev.AddBytes(b)
		}

		n++

		// We first eat spaces

		p.Space()

		co := p.NextByteIs(',')

		if co {
			p.Space()
			p.ev.SetLevel(i)
		} else {
			p.ev.Inc()
		}
	}
}

/* Group

   Group ::= '(' Space? Sequence?  Space? ')'
*/

func (p *Parser) Group() (bool, error) {

	if !p.NextByteIs('(') {
		return false, nil
	}

	i := p.ev.Level()

	p.Space()

	p.Sequence()

	p.Space()

	if !p.NextByteIs(')') {
		return false, errors.New("Missing )")
	}

	// Level before and after a group is the same
	p.ev.SetLevel(i)
	return true, nil
}

// scalar ::= quoted | string
//
func (p *Parser) Scalar() []byte {
	b := p.Quoted()
	if b != nil {
		return b
	}
	return p.String()
}

/* Comment

   Anything from # up to the end of the line.

   BUG(): Special cases: #?, #{
*/
func (p *Parser) Comment() bool {
	c := p.Read()
	if c == '#' {
		for {
			c = p.Read()
			if c == 13 {
				c = p.Read()
				if c != 10 {
					p.Unread()
				}
				break
			}
			if c == 10 {
				break
			}
		}
		return true
	}
	p.Unread()
	return false
}

// String is a concatenation of characters that are > 0x20
// and are not '(', ')', ',', and do not begin with '#'.
//
// NOTE: '#' is allowed inside a string. For '#' to start
// a comment it must be preceeded by break or space, or come
// after a closing ')'.
func (p *Parser) String() []byte {

	c := p.Read()

	if !IsTextChar(c) || c == '#' {
		p.Unread()
		return nil
	}

	buf := make([]byte, 1, 16)
	buf[0] = byte(c)

	for {
		c = p.Read()
		if !IsTextChar(c) {
			p.Unread()
			break
		}
		buf = append(buf, byte(c))
	}

	return buf
}

// Quoted string.
//
// a "quoted string"
//   "text with
//   some
// newlines"
//
func (p *Parser) Quoted() []byte {

	cs := p.Read()
	if cs != '"' && cs != '\'' {
		p.Unread()
		return nil
	}

	buf := make([]byte, 0, 16)

	// p.lastnl is the indentation of this quoted string
	lnl := p.lastnl

	/* Handle \", \', and spaces after NL */
	for {
		c := p.Read()
		if c == cs {
			break
		}

		buf = append(buf, byte(c))

		if c == 10 {
			_, n := p.Space()
			// There are n spaces. Skip lnl spaces and add rest.
			for ; n-lnl > 0; n-- {
				buf = append(buf, ' ')
			}
		} else if c == '\\' {
			c = p.Read()
			if c != '"' && c != '\'' {
				buf = append(buf, '\\')
			}
			buf = append(buf, byte(c))
		}
	}

	// May have zero length
	return buf
}

// Block ::= '\\' NL LINES_OF_TEXT
//
func (p *Parser) Block() string {

	var c int

	c = p.Read()
	if c != '\\' {
		p.Unread()
		return ""
	}

	c = p.Read()
	if c != 10 && c != 13 {
		p.Unread()
		p.Unread()
		return ""
	}

	// read lines until indentation is >= to upper level.
	i := p.ind[p.ev.Level()-1]

	u, ns := p.Space()

	if u && ns == 0 {
		println("Non uniform space at beginning of block at line", p.line)
		panic("")
	}

	buffer := &bytes.Buffer{}

	j := ns

	for {
		if j <= i {
			p.spaces = j /// XXX: unread spaces!
			break
		}

		// Adjust indentation if less that initial
		if j < ns {
			ns = j
		}

		// Read bytes until end of line
		for {
			c = p.Read()

			buffer.WriteByte(byte(c))
			if c == 13 {
				continue
			}

			if c == 10 || p.End() {
				break
			}
		}

		_, j = p.Space()
	}

	// Remove trailing NL
	if c == 10 {
		if buffer.Len() > 0 {
			buffer.Truncate(buffer.Len() - 1)
		}
	}

	return buffer.String()
}

// Break is NL, CR or CR+NL
//
func (p *Parser) Break() bool {
	c := p.Read()
	if c == 13 {
		c = p.Read()
		if c != 10 {
			p.Unread()
		}
		return true
	}
	if c == 10 {
		return true
	}
	p.Unread()
	return false
}

// Space is (0x20|0x09)+. It returns a boolean indicating
// if space has been found, and an integer indicating
// how many spaces, iff uniform (either all 0x20 or 0x09)
//
func (p *Parser) Space() (bool, int) {

	// The Block() production eats to many spaces trying to
	// detect the end of it. They are saved in p.spaces.
	if p.spaces > 0 {
		i := p.spaces
		p.spaces = 0
		return true, i
	}

	c := p.Read()
	if c != 32 && c != 9 {
		p.Unread()
		return false, 0
	}

	n := 1
	/* We keep 'c' to tell us what spaces will count as uniform. */

	for {
		cs := p.Read()
		if cs != 32 && cs != 9 {
			p.Unread()
			break
		}
		if n != 0 && cs == c {
			n++
		} else {
			n = 0
		}
	}

	return true, n
}

// end returns true if it can read an end of stream from the Parser input
// stream.
//
// end < stream > bool
func (p *Parser) End() bool {
	c := p.Read()
	if c < 32 && c != 9 && c != 10 && c != 13 {
		return true
	}
	p.Unread()
	return false
}

func (p *Parser) Newline() bool {
	c := p.Read()
	if c == '\r' {
		c = p.Read()
	}

	if c == '\n' {
		return true
	}

	p.Unread()
	return false
}

// token reads from the Parser input stream and returns
// a token or nil. A token is defined as a sequence of
// letters and/or numbers and/or _.
//
// Examples of tokens:
//  _a
//  1
//  143lasd034
//
func (p *Parser) Token() []byte {

	c := p.Read()

	if !IsTokenChar(c) {
		p.Unread()
		return nil
	}

	buf := make([]byte, 1, 16)
	buf[0] = byte(c)

	for {
		c = p.Read()
		if !IsTokenChar(c) {
			p.Unread()
			break
		}
		buf = append(buf, byte(c))
	}

	return buf
}

// index ::= '[' token ']'
//
/*
func (p *Parser) pIndex() ([]byte, error) {

	if !p.nextByteIs('[') {
		return nil, nil
	}

	p.Space()
	s := p.Token()
	p.Space()

	if !p.nextByteIs(']') {
		return nil, errors.New("Missing ] in index")
	}

	if len(s) == 0 {
		return nil, errors.New("Empty index")
	}
	return s, nil
}
*/

// selector reads from the Parser input stream and returns
// a selector token, nil or an error. An empty selector is
// legal and is represented by one space.
//
// selector ::= '{' token? '}'
//
/*
func (p *Parser) pSelector() ([]byte, error) {

	if !p.nextByteIs('{') {
		return nil, nil
	}

	p.pSpace()
	s := p.pToken()
	p.pSpace()

	if !p.nextByteIs('}') {
		return nil, errors.New("Missing } in selector")
	}

	// Return one space to indicate an empty selector
	if len(s) == 0 {
		return []byte(" "), nil
	}
	return s, nil
}
*/

func (p *Parser) Number() []byte {

	c := p.Read()

	if !IsDigit(c) {
		if c != '-' {
			p.Unread()
			return nil
		}
		d := p.Read()
		if !IsDigit(d) {
			p.Unread()
			p.Unread()
			return nil
		}
		p.Unread()
	}

	buf := make([]byte, 1, 16)
	buf[0] = byte(c)

	for {
		c = p.Read()
		if !IsDigit(c) && c != '.' {
			p.Unread()
			break
		}
		buf = append(buf, byte(c))
	}

	return buf
}

func (p *Parser) Operator() []byte {

	c := p.Read()

	if !IsOperatorChar(c) {
		p.Unread()
		return nil
	}

	buf := make([]byte, 1, 16)
	buf[0] = byte(c)

	for {
		c = p.Read()
		if !IsOperatorChar(c) {
			p.Unread()
			break
		}
		buf = append(buf, byte(c))
	}

	return buf
}

// expression := expr1 (op2 expr1)*
//
func (p *Parser) Expression() bool {
	if !p.UnaryExpression() {
		return false
	}

	for {
		p.Space()
		b := p.Operator()
		if b != nil {
			p.ev.AddBytes(b)
		} else {
			return true
		}
		p.Space()
		if !p.UnaryExpression() {
			return false // error
		}
		p.Space()
	}
}

// expr1 := cpath | constant | op1 cpath | op1 constant | '(' expr ')' | op1 '(' expr ')'
//
func (p *Parser) UnaryExpression() bool {

	c := p.Read()
	p.Unread()

	if IsLetter(c) {
		p.ev.Add(TYPE_PATH)
		p.ev.Inc()
		p.Path()
		p.ev.Dec()
		return true
	}

	b := p.Number()
	if b != nil {
		p.ev.AddBytes(b)
		return true
	}

	b = p.Quoted()
	if b != nil {
		p.ev.AddBytes(b)
		return true
	}

	b = p.Operator()
	if b != nil {
		p.ev.AddBytes(b)
	}

	if p.NextByteIs('(') {

		p.ev.Add(TYPE_GROUP)
		p.ev.Inc()
		p.Space()
		p.Expression()
		p.Space()
		p.ev.Dec()

		return p.NextByteIs(')')
	}

	return p.Path()
}

func (p *Parser) Text() bool {

	c := p.Read()

	if !IsTemplateTextChar(c) {
		p.Unread()
		return false
	}

	buf := make([]byte, 1, 16)
	buf[0] = byte(c)

	for {
		c := p.Read()
		if !IsTemplateTextChar(c) {

			p.Unread()
			break
		}
		buf = append(buf, byte(c))
	}

	p.ev.AddBytes(buf)
	return true
}

func (p *Parser) Variable() bool {

	c := p.Read()

	if c != '$' {
		p.Unread()
		return false
	}

	c = p.Read()
	if c == '\\' {
		p.ev.Add("$")
		return true
	} else {
		p.Unread()
	}

	i := p.ev.Level()

	c = p.Read()
	if c == '(' || c == '{' {
		p.ev.Add(TYPE_EXPRESSION)
		p.ev.Inc()
		p.Expression()
		p.Space()
		c = p.Read() // Should be ')' or '}'
	} else {
		p.ev.Add(TYPE_PATH)
		p.ev.Inc()
		p.Unread()
		p.Path()
	}

	// Reset the level
	p.ev.SetLevel(i)

	return true

}

// index ::= '[' expression ']'
//
func (p *Parser) Index() bool {

	if !p.NextByteIs('[') {
		return false
	}

	i := p.ev.Level()

	p.ev.Add(TYPE_INDEX)
	p.ev.Inc()

	p.Space()
	p.Expression()
	p.Space()

	if !p.NextByteIs(']') {
		return false // error
	}

	/* Level before and after is the same */
	p.ev.SetLevel(i)
	return true
}

// selector ::= '{' expression? '}'
//
func (p *Parser) Selector() bool {

	if !p.NextByteIs('{') {
		return false
	}

	i := p.ev.Level()

	p.ev.Add(TYPE_SELECTOR)
	p.ev.Inc()

	p.Space()
	p.Expression()
	p.Space()

	if !p.NextByteIs('}') {
		return false // error
	}

	/* Level before and after is the same */
	p.ev.SetLevel(i)
	return true
}

// args ::= '(' space? sequence? space? ')'
//
func (p *Parser) Args() (bool, error) {

	if !p.NextByteIs('(') {
		return false, nil
	}

	i := p.ev.Level()

	p.ev.Add(TYPE_GROUP)
	p.ev.Inc()

	p.Space()
	p.ArgList()
	p.Space()

	if !p.NextByteIs(')') {
		return false, errors.New("Missing )")
	}

	/* Level before and after is the same */
	p.ev.SetLevel(i)
	return true, nil
}

// arglist ::= space? expression space? [, space? expression]* space?
//
// arglist < stream > events
//
// arglist can be empty, then returning false (this fact is not represented
// in the BNF definition).
//
func (p *Parser) ArgList() bool {

	something := false

	for {
		p.Space()

		p.ev.Add(TYPE_EXPRESSION)
		p.ev.Inc()
		if !p.Expression() {
			p.ev.Dec()
			p.ev.Delete()
			return something
		}
		p.ev.Dec()
		something = true

		p.Space()
		p.NextByteIs(',')
	}
}

// TokenList ::= token [, token]*
func (p *Parser) TokenList() {

	comma := false

	for {
		p.Space()

		if comma && !p.NextByteIs(',') {
			return
		} else {
			p.Space()
		}

		s := p.Token()
		if len(s) == 0 {
			return
		}

		p.ev.AddBytes(s)
		comma = true
	}
}

// Template ::= (Text | Variable | Format)*
//
// Some variables + text form $for $end and $if $else $end
// constructs.
//
// - First pass produces a list of text and variables
// - Second pass structures $for and $if
func (p *Parser) Template() {
	for {
		if !p.Text() && !p.Variable() {
			break
		}
	}
}
