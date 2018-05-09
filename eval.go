// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

// import "fmt"

// Eval takes a parsed expression and evaluates it
// in the context of the current graph.
func (g *Graph) Eval(e *Graph) (interface{}, error) {

	switch e.ThisString() {
	case TypePath:
		return g.evalPath(e)
	case TypeExpression:
		return g.evalExpression(e)
	}

	// A complex object that is not a path or expression: return as is.
	if e.Len() != 0 {
		return e, nil
	}

	// A constant: return in its normalizad native form
	// either: int64, float64, string, bool or []byte
	return e.ThisScalar(), nil
}

// EvalBool takes a parsed expression and evaluates it in the context of the
// current graph, and converts the result to a boolean.
func (g *Graph) evalBool(e *Graph) bool {
	i, err := g.Eval(e)
	if err != nil {
		return false
	}
	b, _ := _boolf(i)
	return b
}

// GetPath <-> EvalPath
func (g *Graph) getPath(p *Graph) (*Graph, error) {

	if p.Len() == 0 || g == nil {
		return nil, ErrInvalidArgs
	}

	// fmt.Printf("%s\n", p.Show())

	ctx := g            // Current context node
	var ctxPrev *Graph  // Previous context node
	var elemPrev string // Previous path element searched in the context node

	for i := 0; i < len(p.Out); i++ {

		if ctx == nil {
			break
		}

		pathNode := p.Out[i]
		pathElement := pathNode.ThisString()

		switch pathElement {

		case TypeIndex:

			// must evaluate to an integer
			ix := pathNode.index(g)

			if ix < 0 {
				return nil, ErrInvalidIndex
			}

			r := New()
			r.Add(ctx.GetAt(ix))
			ctxPrev = ctx
			ctx = r

		case TypeSelector:

			if ctxPrev == nil || len(elemPrev) == 0 {
				return nil, ErrInvalidIndex
			}

			r := New()
			ix := pathNode.index(g) + 1 // 0 is {}, {n} becomes ix = n+1

			if ix < 0 {
				return nil, ErrInvalidIndex
			} else if ix == 0 {
				// This case is {}, thus return all ocurrences of the token just before
				r.addEqualNodes(ctxPrev, elemPrev, false)
			} else {
				// of all the nodes with name elemPrev, select the ix_th.
				for _, nn := range ctxPrev.Out {
					if nn.ThisString() == elemPrev {
						ix--
						if ix == 0 {
							r.AddNodes(nn)
							break
						}
					}
				}

			}
			ctxPrev = ctx
			ctx = r

		case TypeArguments:
			// We have hit an argument list of a function
			if ctx.Len() > 0 {
				itf, err := g.function(p, ctx.GetAt(0).This)
				if err != nil {
					return nil, err
				}
				var ok bool
				ctxPrev = ctx
				ctx, ok = itf.(*Graph)
				if !ok {
					return nil, ErrFunctionNoGraph
				}
			} else {
				return nil, ErrNotFound
			}

		case TypeGroup:
			// The expression is evaluated and used as path element
			itf, err := g.Eval(pathNode.Out[0])
			if err != nil {
				return nil, err
			}
			str := _string(itf)

			if len(str) == 0 {
				return nil, nil // expr does not evaluate to a string
			}
			pathElement = str
			// [!] .().
			fallthrough

		default:
			ctxPrev = ctx
			ctx = ctx.Node(pathElement)
			elemPrev = pathElement

			if ctx == nil {
				if ctxPrev.Len() > 0 {
					itf, err := g.function(p, ctxPrev.GetAt(0).This)
					if err != nil {
						return nil, err
					}
					var ok bool
					ctx, ok = itf.(*Graph)
					if !ok {
						return nil, ErrFunctionNoGraph
					}
				}
			}
		}
	}

	return ctx, nil
}

// evalPath traverses g following a path p. The path needs to be previously converted
// to a Graph with NewPath().
//
func (g *Graph) evalPath(p *Graph) (interface{}, error) {

	// fmt.Printf("evalPath\n%s\n%s\n", p.Show(), g.Show())

	if p.Len() == 0 || g == nil {
		return nil, ErrInvalidArgs
	}

	ctx := g            // Current context node
	var ctxPrev *Graph  // Previous context node
	var elemPrev string // Previous path element searched in the context node
	var addRoot bool

	for i := 0; i < len(p.Out); i++ {

		if ctx == nil {
			break
		}

		pathNode := p.Out[i]
		pathElement := pathNode.ThisString()
		addRoot = false

		switch pathElement {

		case TypeIndex:

			// must evaluate to an integer
			ix := pathNode.index(g)
			if ix < 0 {
				return nil, ErrInvalidIndex
			}

			ctxPrev = ctx
			ctx = ctx.GetAt(ix)
			addRoot = true

		case TypeSelector:

			if ctxPrev == nil || len(elemPrev) == 0 {
				return nil, ErrInvalidIndex
			}

			r := New()
			ix := pathNode.index(g) + 1 // 0 is {}, {n} becomes ix = n+1

			if ix < 0 {
				return nil, ErrInvalidIndex
			} else if ix == 0 {
				// This case is {}, thus return all ocurrences of the token just before
				r.addEqualNodes(ctxPrev, elemPrev, false)
			} else {
				// of all the nodes with name elemPrev, select the ix_th.
				for _, nn := range ctxPrev.Out {
					if nn.ThisString() == elemPrev {
						ix--
						if ix == 0 {
							r.AddNodes(nn)
							break
						}
					}
				}

			}
			ctxPrev = ctx
			ctx = r

		case "_len":
			return ctx.Len(), nil

		case "_this":
			return ctx, nil

		case "_thisString":
			return ctx.ThisString(), nil

		case "_string":
			return ctx.String(), nil

		case TypeArguments:
			// We have hit an argument list of a function
			if ctx.Len() > 0 {
				itf, err := g.function(p, ctx.GetAt(0).This)
				if err != nil {
					return nil, err
				}
				var ok bool
				ctxPrev = ctx
				ctx, ok = itf.(*Graph)
				if !ok {
					return itf, nil
				}
			} else {
				return nil, nil
			}

		case TypeGroup:
			// The expression is evaluated and used as path element
			itf, err := g.Eval(pathNode.Out[0])
			if err != nil {
				return nil, err
			}
			str := _string(itf)

			if len(str) == 0 {
				return nil, nil // expr does not evaluate to a string
			}
			pathElement = str
			// [!] .().
			fallthrough

		default:
			ctxPrev = ctx
			ctx = ctx.Node(pathElement)
			elemPrev = pathElement

			if ctx == nil {
				if ctxPrev.Len() > 0 {
					itf, err := g.function(p, ctxPrev.GetAt(0).This)
					if err != nil {
						return nil, err
					}
					var ok bool
					ctx, ok = itf.(*Graph)
					if !ok {
						return itf, nil
					}
					// fmt.Printf("evalPath function %s %d %d %d\n", pathElement, i, p.Len(), len(p.Out))
					return itf, nil
				}
			}

		}
	}

	if addRoot {
		r := New()
		r.This = "["
		r.Add(ctx)
		ctx = r
	}

	return _simplify(ctx), nil
}

// EvalExpression evaluates expressions (!e)
// g can have native types (other things than strings), but
// p only []byte or string
//
func (g *Graph) evalExpression(p *Graph) (interface{}, error) {

	// Return nil and empty strings as is
	if p.This == nil {
		return nil, ErrNilReceiver
	}

	s := p.ThisString()

	if len(s) == 0 {
		return "", nil
	}

	// first check if it is a number because it can have an operatorChar
	// in front: the minus sign
	if isNumber(s) {
		return p.ThisNumber(), nil
	}

	switch s {
	case "!":
		// Unary expression !expr
		return !g.evalBool(p.Out[0]), nil
	case TypeExpression:
		return g.evalExpression(p.GetAt(0))
	case TypePath:
		return g.evalPath(p)
	case TypeGroup:
		// TODO expression list (could also be OGDL flow!)
		r := New(TypeGroup)
		for _, expr := range p.Out {
			itf, err := g.evalExpression(expr)
			if err == nil {
				r.Add(itf)
			}
		}
		return r, nil
	case TypeString:
		if p.Len() == 0 {
			return "", nil
		}
		return p.GetAt(0).ThisString(), nil

	}

	c := s[0]

	// [!] Operator should be identified. Operators written as strings are
	// missinterpreted.
	if isOperatorChar(c) {
		if len(s) <= 2 {
			if len(s) == 1 || isOperatorChar(s[1]) {
				return g.evalBinary(p), nil
			}
		}
	}

	if c == '"' || c == '\'' {
		return s, nil
	}

	if IsLetter(rune(c)) {
		if s == "false" {
			return false, nil
		}
		if s == "true" {
			return true, nil
		}
		return s, nil
	}

	return p, nil
}

func (g *Graph) evalBinary(p *Graph) interface{} {

	n1 := p.Out[0]

	i2, err := g.evalExpression(p.Out[1])
	if err != nil {
		return err // ?
	}

	switch p.ThisString() {
	case "=":
		return g.assign(n1, i2, '=')
	case "+=":
		return g.assign(n1, i2, '+')
	case "-=":
		return g.assign(n1, i2, '-')
	case "*=":
		return g.assign(n1, i2, '*')
	case "/=":
		return g.assign(n1, i2, '/')
	case "%=":
		return g.assign(n1, i2, '%')
	}

	i1, err := g.evalExpression(n1)
	if err != nil {
		return err // ?
	}

	switch p.ThisString() {

	case "+":
		return calc(i1, i2, '+')
	case "-":
		return calc(i1, i2, '-')
	case "*":
		return calc(i1, i2, '*')
	case "/":
		return calc(i1, i2, '/')
	case "%":
		return calc(i1, i2, '%')

	case "==":
		return compare(i1, i2, '=')
	case ">=":
		return compare(i1, i2, '+')
	case "<=":
		return compare(i1, i2, '-')
	case "!=":
		return compare(i1, i2, '!')
	case ">":
		return compare(i1, i2, '>')
	case "<":
		return compare(i1, i2, '<')

	case "&&":
		return logic(i1, i2, '&')
	case "||":
		return logic(i1, i2, '|')

	}

	return nil
}

// int* | float* | string
// first element determines type
func compare(v1, v2 interface{}, op int) bool {

	i1, ok := _int64(v1)

	if ok {
		i2, ok := _int64f(v2)
		if !ok {
			return false
		}

		switch op {
		case '=':
			return i1 == i2
		case '+':
			return i1 >= i2
		case '-':
			return i1 <= i2
		case '>':
			return i1 > i2
		case '<':
			return i1 < i2
		case '!':
			return i1 != i2
		}
		return false
	}

	f1, ok := _float64(v1)
	if ok {
		f2, ok := _float64f(v2)
		if !ok {
			return false
		}
		switch op {
		case '=':
			return f1 == f2
		case '+':
			return f1 >= f2
		case '-':
			return f1 <= f2
		case '>':
			return f1 > f2
		case '<':
			return f1 < f2
		case '!':
			return f1 != f2
		}
		return false
	}

	s1 := _string(v1)
	s2 := _string(v2)

	switch op {
	case '=':
		return s1 == s2
	case '!':
		return s1 != s2
	}
	return false
}

func logic(i1, i2 interface{}, op int) bool {

	b1, ok1 := _boolf(i1)
	b2, ok2 := _boolf(i2)

	if !ok1 || !ok2 {
		return false
	}

	switch op {
	case '&':
		return b1 && b2
	case '|':
		return b1 || b2
	}

	return false
}

// assign modifies the context graph
func (g *Graph) assign(p *Graph, v interface{}, op int) interface{} {

	if op == '=' {
		return g.set(p, v)
	}

	// if p doesn't exist, just set it to the value given
	left, _ := g.getPath(p)
	if left != nil {
		return g.set(p, calc(left.Out[0].This, v, op))
	}

	switch op {
	case '+':
		return g.set(p, v)
	case '-':
		return g.set(p, calc(0, v, '-'))
	case '*':
		return g.set(p, 0)
	case '/':
		return g.set(p, "infinity")
	case '%':
		return g.set(p, "undefined")
	}

	return nil
}

// calc: int64 | float64 | string
func calc(v1, v2 interface{}, op int) interface{} {

	i1, ok := _int64(v1)
	i2, ok2 := _int64(v2)

	var ok3, ok4 bool
	var i3, i4 float64

	if !ok {
		i3, ok3 = _float64(v1)
	}
	if !ok2 {
		i4, ok4 = _float64(v2)
	}

	if ok && ok2 {
		switch op {
		case '+':
			return i1 + i2
		case '-':
			return i1 - i2
		case '*':
			return i1 * i2
		case '/':
			return i1 / i2
		case '%':
			return i1 % i2
		}
	}
	if ok3 && ok4 {
		switch op {
		case '+':
			return i3 + i4
		case '-':
			return i3 - i4
		case '*':
			return i3 * i4
		case '/':
			return i3 / i4
		case '%':
			return int(i3) % int(i4)
		}
	}
	if ok && ok4 {
		i3 = float64(i1)
		switch op {
		case '+':
			return i3 + i4
		case '-':
			return i3 - i4
		case '*':
			return i3 * i4
		case '/':
			return i3 / i4
		case '%':
			return i1 % int64(i4)
		}
	}
	if ok3 && ok2 {
		i4 = float64(i2)
		switch op {
		case '+':
			return i3 + i4
		case '-':
			return i3 - i4
		case '*':
			return i3 * i4
		case '/':
			return i3 / i4
		case '%':
			return int64(i3) % i2
		}
	}

	if op != '+' {
		return nil
	}

	return _string(v1) + _string(v2)
}

// TODO Armonize with ThisScalar, Scalar
func (g *Graph) scalar(numeric bool) interface{} {

	if g == nil || g.Len() == 0 {
		return nil
	}
	/*
		if g.Len() > 1 || g.Out[0].Len() != 0 {
			n := New()
			n.AddNodes(g)
			return n
		}
	*/
	itf := g.Out[0].This

	if numeric {
		n := number(itf)
		if n != nil {
			return n
		}
	}

	// If it can be parsed as a bool, return it.
	b, ok := _boolf(itf)
	if ok {
		return b
	}

	// Else return as is.
	return itf
}

func (n *Graph) index(g *Graph) int {
	if n.Len() == 0 {
		return -1
	}

	itf, err := g.evalExpression(n.Out[0])
	if err != nil {
		return -2
	}
	ix, ok := _int64(itf)
	if !ok || ix < 0 {
		return -3
	}
	return int(ix)
}
