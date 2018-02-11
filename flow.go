package ogdl

import (
	"bytes"
	"errors"
)

// NewPath takes a Unicode string representing an OGDL path, parses it and
// returns it as a Graph object.
//
// It also parses extended paths, as those used in templates, which may have
// argument lists.
func NewPath(s string) *Graph {
	p := NewParser(bytes.NewBuffer([]byte(s)))
	p.Path()
	g := p.Graph()
	g.This = TypePath
	return g
}

// Index ::= '[' expression ']'
func (p *Parser) Index() bool {

	if p.PeekByte() != '[' {
		return false
	}
	p.Byte()

	i := p.ev.Level()

	p.ev.Add(TypeIndex)
	p.ev.Inc()

	p.Space()
	p.Expression()
	p.Space()

	if p.PeekByte() != ']' {
		return false // error
	}
	p.Byte()

	// Level before and after is the same
	p.ev.SetLevel(i)
	return true
}

// Selector ::= '{' expression? '}'
func (p *Parser) Selector() bool {

	if p.PeekByte() != '{' {
		return false
	}
	p.Byte()

	i := p.ev.Level()

	p.ev.Add(TypeSelector)
	p.ev.Inc()

	p.Space()
	p.Expression()
	p.Space()

	if p.PeekByte() != '}' {
		return false // error
	}
	p.Byte()

	// Level before and after is the same
	p.ev.SetLevel(i)
	return true
}

// ArgList ::= space? expression space? [, space? expression]* space?
//
// arglist < stream > events
//
// arglist can be empty, then returning false (this fact is not represented
// in the BNF definition).
//
func (p *Parser) ArgList() bool {

	something := false

	level := p.ev.Level()
	defer p.ev.SetLevel(level)

	for {
		p.WhiteSpace()

		p.ev.Add(TypeExpression)
		p.ev.Inc()

		if !p.Expression() {

			p.ev.Dec()
			p.ev.Delete()
			return something
		}
		p.ev.Dec()
		something = true

		p.WhiteSpace()

		c := p.PeekByte()

		if c == ')' {
			return true
		}
		if c == ',' {
			p.Byte()
			p.ev.SetLevel(level)
		} else {
			p.ev.Inc()
		}

	}
}

// Args ::= '(' space? sequence? space? ')'
func (p *Parser) Args(dot bool) (bool, error) {

	if p.PeekByte() != '(' {
		return false, nil
	}
	p.Byte()

	i := p.ev.Level()

	if dot {
		p.ev.Add(TypeGroup)
	} else {
		p.ev.Add(TypeArguments)
	}
	p.ev.Inc()

	p.WhiteSpace()
	p.ArgList()
	p.WhiteSpace()

	if p.PeekByte() != ')' {
		return false, errors.New("missing )")
	}
	p.Byte()

	// Level before and after is the same
	p.ev.SetLevel(i)
	return true, nil
}

// Expression := expr1 (op2 expr1)*
//
func (p *Parser) Expression() bool {

	if !p.UnaryExpression() {
		return false
	}

	for {
		p.Space()
		b, ok := p.Operator()
		if ok {
			p.ev.Add(b)
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

// UnaryExpression := cpath | constant | op1 cpath | op1 constant | '(' expr ')' | op1 '(' expr ')'
//
func (p *Parser) UnaryExpression() bool {

	c := p.PeekByte()

	if IsLetter(rune(c)) {
		p.ev.Add(TypePath)
		p.ev.Inc()
		p.Path()
		p.ev.Dec()
		return true
	}

	b, ok := p.Number()
	if ok {
		p.ev.Add(b)
		return true
	}

	b, ok, _ = p.Quoted(0)
	if ok {
		p.ev.Add(TypeString)
		p.ev.Inc()
		p.ev.Add(b)
		p.ev.Dec()
		return true
	}

	b, ok = p.Operator()
	if ok {
		p.ev.Add(b)
	}

	if p.PeekByte() == '(' {
		p.Byte() // Consume the '('
		p.ev.Add(TypeGroup)
		p.ev.Inc()
		p.Space()
		p.Expression()
		p.Space()
		p.ev.Dec()

		if p.PeekByte() != ')' {
			return false
		}
		p.Byte() // Consume the ')'
		return true
	}

	return p.Path()
}

// Path parses an OGDL path, or an extended path as used in templates.
//
//     path ::= element ('.' element)*
//
//     element ::= token | integer | quoted | group | index | selector
//
//     (Dot optional before Group, Index, Selector)
//
//     group := '(' Expression [[,] Expression]* ')'
//     index := '[' Expression ']'
//     selector := '{' Expression '}'
//
// The OGDL parser doesn't need to know about Unicode. The character
// classification relies on values < 127, thus in the ASCII range,
// which is also part of Unicode.
//
// Note: On the other hand it would be stupid not to recognize for example
// Unicode quotation marks if we know that we have UTF-8. But when do we
// know for sure?
func (p *Parser) Path() bool {

	var b string
	var begin = true
	var anything = false
	var ok bool
	//var err error

	// dot keeps track of just read dots. This is used in Args(), to
	// distinguish between a(b) and a.(b)
	dot := true

	for {

		// Expect: token | quoted | index | group | selector | dot,
		// or else we abort.

		// A dot is requiered before a token or quoted, except at
		// the beginning

		if !begin {
			c, _ := p.Byte()

			if c != '.' {
				dot = false
				p.UnreadByte()

				// If not [, {, (, break
				if c != '[' && c != '(' && c != '{' {
					break
				}
			} else {
				dot = true
			}
		}

		begin = false

		b, ok, _ = p.Quoted(0)
		if ok {
			p.ev.Add(b)
			anything = true
			continue
		}

		b, ok = p.Token8()

		if ok {
			p.ev.Add(b)
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

		ok, _ = p.Args(dot)
		if ok {
			anything = true
			continue
		}

		break
	}

	return anything
}
