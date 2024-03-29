// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bytes"
)

// NewTemplate parses a text template given as a string and converts it to a Graph.
// Templates have fixed and variable parts. Variables all begin with '$'.
//
// A template is a text file in any format: plain text, HTML, XML, OGDL or
// whatever. The dolar sign acts as an escape character that switches from the
// text to the variable plane. Parsed templates are converted back to text
// by evaluating the variable parts against a Graph object, by means of the
// Process() method.
//
// Template grammar
//
//     template ::= ( text | variable )*
//
//     variable ::= ('$' path) | ('$' '(' expression ')') | ('$' '{' expression '}')
//     path ::= as defined in path.go
//     expression ::= as defined in expression.go
//
// Some variables act as directives: $if, $else, $end, $for, $break.
//
//    $if(expression)
//    $else
//    $end
//
//    $for(destPath,sourcepath)
//      $break
//    $end
//
func NewTemplate(s string) *Graph {
	p := NewParser(bytes.NewBuffer([]byte(s)))
	p.Template()

	t := p.Graph()
	t.This = TypeTemplate
	t.ast()
	t.simplify()

	g := New("")
	t.flow(g, g, 0)
	return g
}

// NewTemplateFromBytes has the same function as NewTemplate except that the input stream
// is a byte array.
func NewTemplateFromBytes(b []byte) *Graph {
	p := NewParser(bytes.NewBuffer(b))
	p.Template()

	t := p.Graph()
	t.This = TypeTemplate
	t.ast()
	t.simplify()

	g := New("")
	t.flow(g, g, 0)
	return g
}

// Process processes the parsed template, returning the resulting text in a byte array.
// The variable parts are resolved out of the Graph given.
func (g *Graph) Process(ctx *Graph) []byte {

	buffer := &bytes.Buffer{}

	g.process(ctx, buffer)

	return buffer.Bytes()
}

func (g *Graph) process(c *Graph, buffer *bytes.Buffer) bool {

	if g == nil || g.Out == nil {
		return false
	}

	yes := false

	for _, n := range g.Out {
		s := n.ThisString()

		switch s {
		case TypePath:
			i, _ := c.Eval(n)
			buffer.WriteString(_text(i))

		case TypeExpression:
			// Silent evaluation
			c.Eval(n)
		case TypeIf:
			// evaluate the expression
			yes = c.evalBool(n.GetAt(0).GetAt(0))
			// if true, evaluate the template part
			if yes {
				n.GetAt(1).process(c, buffer)
			}
		case TypeElseIf:
			if !yes {
				// evaluate the expression
				yes = c.evalBool(n.GetAt(0).GetAt(0))
				// if true, evaluate the template part
				if yes {
					n.GetAt(1).process(c, buffer)
				}
			}
		case TypeElse:
			// if there was a previous if or elseif evaluating to false:
			if !yes {
				n.GetAt(0).process(c, buffer)
			}
		case TypeFor:
			// The first subnode (of !g) is a path
			// The second is an expression evaluating to a list of elements

			i, _ := c.Eval(n.GetAt(0).GetAt(1))

			// fmt.Printf("-------\nfor (expr)\n-------\n%s\n\n", n.GetAt(0).GetAt(1).Show())

			// Check that i is iterable
			gi, ok := i.(*Graph)
			if !ok || gi == nil || gi.Len() == 0 {
				continue
			}

			// IMPORTANT: in paths that end with an index the following step is needed.
			// Indexes in paths return a '[' root on top of the result which must
			// be removed in order to reach the iterable level.
			if gi.ThisString() == "[" {
				gi = gi.Out[0]
			}

			// fmt.Printf("-------\nfor (evaluated)\n-------\n%s\n\n", gi.Show())

			// The third is the subtemplate to travel
			// println ("for type: ",reflect.TypeOf(i).String(), "ok",ok)
			// Assing expression value to path
			// XXX if not Graph

			varname := n.GetAt(0).GetAt(0).GetAt(0).String()
			c.Delete(varname)
			it := c.Add(varname)

			for _, ee := range gi.Out {

				it.Out = nil
				it.Add(ee)

				brk := n.GetAt(1).process(c, buffer)
				if brk {
					break
				}
			}
		case TypeBreak:
			return true

		default:
			buffer.WriteString(n.ThisString())
		}
	}
	return false
}

// simplify converts !p TYPE in !TYPE for keywords if, end, else, elseif, for and break.
func (g *Graph) simplify() {

	if g == nil {
		return
	}

	for _, node := range g.Out {
		if TypePath == node.ThisString() {
			s := node.GetAt(0).ThisString()

			switch s {
			case "if":
				node.This = TypeIf
				node.DeleteAt(0)
			case "end":
				node.This = TypeEnd
				node.DeleteAt(0)
			case "else":
				node.This = TypeElse
				node.DeleteAt(0)
			case "elseif":
				node.This = TypeElseIf
				node.DeleteAt(0)
			case "for":
				node.This = TypeFor
				node.DeleteAt(0)
			case "break":
				node.This = TypeBreak
				node.DeleteAt(0)
			}
		}
	}

}

// flow nests 'if' and 'for' loops.
// new version from 1/2024
//
// It creates 2 nodes (!a and !t) below 'if' and 'elseif', and 1 node (!t)
// below 'else'
func (g *Graph) flow(h, prev *Graph, start int) int {

	var i int

	for i = start; i < g.Len(); i++ {
		node := g.Out[i]
		s := node.ThisString()

		switch s {

		case TypeIf, TypeFor:
			h.Add(node)
			hh := node.Add("!t")
			i = g.flow(hh, h, i+1)

		case TypeElse, TypeElseIf:
			// Add 'else' and 'elseif' at the same level as 'if' (one level up)
			prev.Add(node)
			// Add the rest of the nodes at this level to this 'template' subnode
			h = node.Add("!t")

		case TypeEnd:
			return i

		default:
			h.Add(node)
		}

	}

	return i
}

/* flow nests 'if' and 'for' loops.
func (g *Graph) flow_() {
	n := 0
	var nod *Graph

	for i := 0; i < g.Len(); i++ {

		node := g.Out[i]
		s := node.ThisString()

		if s == TypeIf || s == TypeFor {
			n++
			if n == 1 {
				nod = node.Add(TypeTemplate)
				continue
			}
		}

		if s == TypeElse || s == TypeElseIf {
			if n == 1 {
				nod.flow_()
				nod = node
				continue
			}
		}

		if s == TypeEnd {
			n--
			if n == 0 {
				nod.flow_()
				g.DeleteAt(i)
				i--
				continue
			}
		}

		if n > 0 {
			nod.Add(node)
			g.DeleteAt(i)
			i--
		}
	}

}
*/

// Template ::= (Text | Variable)*
func (p *Parser) Template() {
	for {
		if !p.Text() && !p.Variable() {
			break
		}
	}
}

// Text returns the next text part of the template (until it finds a variable)
func (p *Parser) Text() bool {

	s, b := p.TemplateText()

	if b {
		p.ev.Add(s)
		return true
	}
	return false
}

// Variable parses variables in a template. They begin with $.
func (p *Parser) Variable() bool {

	c, _ := p.Byte()

	if c != '$' {
		p.UnreadByte()
		return false
	}

	c, _ = p.Byte()
	if c == '\\' {
		p.ev.Add("$")
		return true
	}

	p.UnreadByte()

	i := p.ev.Level()

	c, _ = p.Byte()
	if c == '(' {
		p.ev.Add(TypeExpression)
		p.ev.Inc()
		p.Expression()
		p.Space()
		p.Byte() // Should be ')'
	} else {
		p.ev.Add(TypePath)
		p.ev.Inc()
		if c != '{' {
			p.UnreadByte()
		} else {
			p.Space()
		}
		p.Path()

		if c == '{' {
			p.Space()
			p.Byte() // Should be '}'
		}
	}

	// Reset the level
	p.ev.SetLevel(i)

	return true

}
