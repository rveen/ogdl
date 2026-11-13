package ogdl

import (
	"errors"
	"strings"
)

// This file implements the evaluation rules of the expression RFC, section 6,
// and path evaluation extended by calls and groups (sections 5 and 8).

// Errors reported while evaluating (Expr RFC 9).
var (
	ErrDivideByZero     = errors.New("division by zero")
	ErrNotANumber       = errors.New("operand is not a number")
	ErrNotComparable    = errors.New("operands are not comparable")
	ErrNotCallable      = errors.New("value is not callable")
	ErrArgCount         = errors.New("wrong number of arguments")
	ErrIncompatibleType = errors.New("incompatible type")
	ErrBadResults       = errors.New("unsupported result signature")
	ErrEvalTooDeep      = errors.New("evaluation exceeds the depth limit")
	ErrNegativeIndex    = errors.New("negative index")
)

// DefaultMaxEvalDepth bounds evaluation recursion, including re-entry through a
// host function (Expr RFC 10).
const DefaultMaxEvalDepth = 200

// evaluator carries the state of one evaluation.
type evaluator struct {
	depth    int
	maxDepth int

	// builtins enables the built-in members of Expr RFC 8.4. They belong to
	// the expression layer, so strict path resolution leaves them off.
	builtins bool

	// vars holds the loop variables in scope (Template RFC 5.2). A variable
	// names the iterated element itself, and is consulted only for the first
	// element of a path, before the children of the context, so it hides a
	// node of the same name without touching the tree.
	vars map[string]*Node
}

// variable returns the element bound to a loop variable, if one is in scope.
func (ev *evaluator) variable(name string) (*Node, bool) {
	n, ok := ev.vars[name]
	return n, ok
}

// EvalString parses and evaluates an expression in the context of this node.
func (n *Node) EvalString(s string) (interface{}, error) {
	e, err := ParseExpr(s)
	if err != nil {
		return nil, err
	}
	return n.Eval(e)
}

// Eval evaluates a parsed expression in the context of this node.
//
// The result is one of the kinds of expression RFC section 2.1: int64, float64,
// string, bool, *Node, a host value, or nil. A path that matches nothing yields
// nil, which is not an error.
func (n *Node) Eval(e Expr) (interface{}, error) {
	ev := &evaluator{maxDepth: DefaultMaxEvalDepth, builtins: true}
	return ev.eval(n, e)
}

func (ev *evaluator) eval(ctx *Node, e Expr) (interface{}, error) {
	ev.depth++
	defer func() { ev.depth-- }()
	if ev.depth > ev.maxDepth {
		return nil, ErrEvalTooDeep
	}

	switch x := e.(type) {
	case *Literal:
		return x.Val, nil

	case *PathExpr:
		v, err := ev.evalPath(ctx, x.Elems)
		if err != nil {
			return nil, err
		}
		return scalarOf(v), nil

	case *Unary:
		return ev.evalUnary(ctx, x)

	case *Binary:
		return ev.evalBinary(ctx, x)
	}

	return nil, ErrInvalidExpr
}

func (ev *evaluator) evalUnary(ctx *Node, x *Unary) (interface{}, error) {
	v, err := ev.eval(ctx, x.X)
	if err != nil {
		return nil, err
	}

	switch x.Op {
	case "!":
		return !toBool(v), nil
	case "-":
		i, f, isInt, ok := toNumber(v)
		if !ok {
			return nil, ErrNotANumber
		}
		if isInt {
			return -i, nil
		}
		return -f, nil
	}
	return nil, ErrInvalidExpr
}

func (ev *evaluator) evalBinary(ctx *Node, x *Binary) (interface{}, error) {
	if isAssign(x.Op) {
		return ev.evalAssign(ctx, x)
	}

	// Boolean operators short-circuit, so the right operand is evaluated only
	// when it is needed (Expr RFC 6.4).
	switch x.Op {
	case "&&":
		l, err := ev.eval(ctx, x.L)
		if err != nil {
			return nil, err
		}
		if !toBool(l) {
			return false, nil
		}
		r, err := ev.eval(ctx, x.R)
		if err != nil {
			return nil, err
		}
		return toBool(r), nil

	case "||":
		l, err := ev.eval(ctx, x.L)
		if err != nil {
			return nil, err
		}
		if toBool(l) {
			return true, nil
		}
		r, err := ev.eval(ctx, x.R)
		if err != nil {
			return nil, err
		}
		return toBool(r), nil
	}

	l, err := ev.eval(ctx, x.L)
	if err != nil {
		return nil, err
	}
	r, err := ev.eval(ctx, x.R)
	if err != nil {
		return nil, err
	}

	switch x.Op {
	case "+", "-", "*", "/", "%":
		return arith(x.Op, l, r)
	case "==", "!=", ">=", "<=", ">", "<":
		return compare(x.Op, l, r)
	}
	return nil, ErrInvalidExpr
}

// evalAssign implements expression RFC section 6.5.
func (ev *evaluator) evalAssign(ctx *Node, x *Binary) (interface{}, error) {
	lhs, ok := x.L.(*PathExpr)
	if !ok {
		return nil, ErrNotAssignable
	}

	// The right operand is evaluated first, so that a failure there leaves the
	// tree untouched (Expr RFC 6.6).
	v, err := ev.eval(ctx, x.R)
	if err != nil {
		return nil, err
	}

	if x.Op != "=" {
		// x op= y is x = x op y, with x evaluated once. The target must
		// already hold something.
		cur, err := ev.evalPath(ctx, lhs.Elems)
		if err != nil {
			return nil, err
		}
		if scalarOf(cur) == nil {
			return nil, ErrTargetMissing
		}
		v, err = arith(strings.TrimSuffix(x.Op, "="), scalarOf(cur), v)
		if err != nil {
			return nil, err
		}
	}

	target, item, err := ev.target(ctx, lhs.Elems)
	if err != nil {
		return nil, err
	}
	assign(target, v, item)
	return v, nil
}

// arith implements expression RFC section 6.2.
func arith(op string, l, r interface{}) (interface{}, error) {
	// A path that matched nothing is not arithmetic. Concatenating it as an
	// empty string would turn a wrong path into a plausible result.
	if scalarOf(l) == nil || scalarOf(r) == nil {
		return nil, ErrNotANumber
	}

	li, lf, lInt, lok := toNumber(l)
	ri, rf, rInt, rok := toNumber(r)

	if lok && rok {
		if lInt && rInt {
			switch op {
			case "+":
				return li + ri, nil
			case "-":
				return li - ri, nil
			case "*":
				return li * ri, nil
			case "/":
				if ri == 0 {
					return nil, ErrDivideByZero
				}
				return li / ri, nil
			case "%":
				if ri == 0 {
					return nil, ErrDivideByZero
				}
				return li % ri, nil
			}
		}

		if op == "%" {
			// Modulo requires two integers (Expr RFC 6.2).
			return nil, ErrNotANumber
		}
		if lInt {
			lf = float64(li)
		}
		if rInt {
			rf = float64(ri)
		}
		switch op {
		case "+":
			return lf + rf, nil
		case "-":
			return lf - rf, nil
		case "*":
			return lf * rf, nil
		case "/":
			if rf == 0 {
				return nil, ErrDivideByZero
			}
			return lf / rf, nil
		}
	}

	// Not both numbers: '+' concatenates, everything else is an error.
	if op == "+" {
		return toString(scalarOf(l)) + toString(scalarOf(r)), nil
	}
	return nil, ErrNotANumber
}

// compare implements expression RFC section 6.3. It is symmetric: both operands
// are examined before a comparison kind is chosen.
func compare(op string, l, r interface{}) (interface{}, error) {
	li, lf, lInt, lok := toNumber(l)
	ri, rf, rInt, rok := toNumber(r)

	if lok && rok {
		if lInt && rInt {
			return orderInt(op, li, ri), nil
		}
		if lInt {
			lf = float64(li)
		}
		if rInt {
			rf = float64(ri)
		}
		return orderFloat(op, lf, rf), nil
	}

	lb, lIsBool := scalarOf(l).(bool)
	rb, rIsBool := scalarOf(r).(bool)
	if lIsBool && rIsBool {
		switch op {
		case "==":
			return lb == rb, nil
		case "!=":
			return lb != rb, nil
		}
		return nil, ErrNotComparable
	}

	return orderString(op, toString(scalarOf(l)), toString(scalarOf(r))), nil
}

func orderInt(op string, a, b int64) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case "<":
		return a < b
	}
	return false
}

func orderFloat(op string, a, b float64) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case "<":
		return a < b
	}
	return false
}

func orderString(op string, a, b string) bool {
	c := strings.Compare(a, b)
	switch op {
	case "==":
		return c == 0
	case "!=":
		return c != 0
	case ">=":
		return c >= 0
	case "<=":
		return c <= 0
	case ">":
		return c > 0
	case "<":
		return c < 0
	}
	return false
}

// resolvePath evaluates a strict path, with no evaluation context. It is what
// Get uses; calls, groups and expression indexes cannot appear.
func resolvePath(root *Node, elems []pathElem) *Node {
	// Strict path resolution: a path addresses nodes, so the built-in members
	// of Expr RFC 8.4 are deliberately not available here.
	ev := &evaluator{maxDepth: DefaultMaxEvalDepth}
	v, err := ev.evalPath(root, elems)
	if err != nil {
		return &Node{}
	}
	if n, ok := v.(*Node); ok && n != nil {
		return n
	}
	if v == nil {
		return &Node{}
	}
	// A host value reached through a strict path: wrap it so the caller still
	// gets a node.
	return &Node{Data: v}
}

// evalPath walks the elements of a path. The cursor is a list of nodes so that
// the empty selector can widen it (path RFC 6.1); a call or a member of a host
// value leaves the tree, and is returned directly.
func (ev *evaluator) evalPath(ctx *Node, elems []pathElem) (interface{}, error) {
	if ctx == nil {
		return nil, nil
	}

	cur := []*Node{ctx}
	var prev []*Node
	name := ""
	set := false

	// host is non-nil once the path has left the tree, e.g. after a call or a
	// struct field access. Further elements resolve against it by reflection.
	var host interface{}

	// isVar is set while the cursor is a loop variable reached by the first
	// element, so that a selector after it addresses the variable itself.
	isVar := false

	for i, e := range elems {
		if host != nil {
			v, err := ev.hostElem(ctx, host, e)
			if err != nil {
				return nil, err
			}
			host = v
			if host == nil {
				return nil, nil
			}
			continue
		}

		var next []*Node

		switch e.kind {
		case pathToken, pathGroup:
			n := e.name
			if e.kind == pathGroup {
				v, err := ev.eval(ctx, e.expr)
				if err != nil {
					return nil, err
				}
				n = toString(scalarOf(v))
				if n == "" {
					return nil, nil
				}
			}
			if i == 0 {
				if v, ok := ev.variable(n); ok {
					cur, prev, name, isVar = []*Node{v}, cur, n, true
					continue
				}
			}
			for _, c := range cur {
				// A child of the tree, else the host value this node stands
				// for, else a built-in (Expr RFC 8.2). The host value is the
				// node's content, which for "obj <hostvalue>" is its child.
				v, ok := ev.resolveMember(c, n)
				if !ok {
					continue
				}
				if child, isNode := v.(*Node); isNode {
					next = append(next, child)
					continue
				}
				host = v
			}
			prev = cur
			name = n
			if host != nil && len(next) == 0 {
				continue
			}

		case pathIndex:
			k := e.n
			if e.expr != nil {
				v, err := ev.eval(ctx, e.expr)
				if err != nil {
					return nil, err
				}
				i64, ok := toInt64(v)
				if !ok {
					return nil, ErrNotANumber
				}
				if i64 < 0 {
					return nil, ErrNegativeIndex
				}
				k = int(i64)
			}
			for _, c := range cur {
				if k < len(c.Out) {
					next = append(next, c.Out[k])
					continue
				}
				// Past the end of the tree: the node may stand for a host
				// slice, array or string.
				if v, err := indexOf(scalarOf(c), k); err == nil && v != nil {
					host = v
				}
			}
			prev = cur
			name = ""
			if len(next) == 0 && host != nil {
				continue
			}

		case pathSelector:
			if name == "" {
				return nil, nil
			}
			if isVar {
				// A variable names one element: it is occurrence 0, and the
				// only member of its set.
				if e.all || e.n == 0 {
					next = cur
				}
				if e.all {
					set = true
				}
				break
			}
			for _, p := range prev {
				k := 0
				for _, child := range p.Out {
					if nameString(child.Data) != name {
						continue
					}
					if e.all {
						next = append(next, child)
					} else if k == e.n {
						next = append(next, child)
						break
					}
					k++
				}
			}
			if e.all {
				set = true
			}

		case pathCall:
			if len(cur) == 0 {
				return nil, nil
			}
			args, err := ev.evalArgs(ctx, e.args)
			if err != nil {
				return nil, err
			}
			v, err := callAny(scalarOf(cur[0]), args)
			if err != nil {
				return nil, err
			}
			if n, ok := v.(*Node); ok {
				next = []*Node{n}
				prev = cur
				break
			}
			host = v
			if host == nil {
				return nil, nil
			}
			continue
		}

		cur = next
		isVar = false
		if len(cur) == 0 && host == nil {
			return nil, nil
		}
	}

	if host != nil {
		return host, nil
	}
	if len(cur) == 0 {
		return nil, nil
	}
	if set {
		return &Node{Out: cur}, nil
	}
	return cur[0], nil
}

// hostElem applies one path element to a value that is no longer part of the
// tree (Expr RFC 8.2).
func (ev *evaluator) hostElem(ctx *Node, host interface{}, e pathElem) (interface{}, error) {
	switch e.kind {
	case pathToken, pathGroup:
		n := e.name
		if e.kind == pathGroup {
			v, err := ev.eval(ctx, e.expr)
			if err != nil {
				return nil, err
			}
			n = toString(scalarOf(v))
		}
		v, ok := ev.resolveMember(host, n)
		if !ok {
			return nil, nil
		}
		return v, nil

	case pathCall:
		args, err := ev.evalArgs(ctx, e.args)
		if err != nil {
			return nil, err
		}
		return callAny(host, args)

	case pathIndex:
		k := e.n
		if e.expr != nil {
			v, err := ev.eval(ctx, e.expr)
			if err != nil {
				return nil, err
			}
			i64, ok := toInt64(v)
			if !ok {
				return nil, ErrNotANumber
			}
			if i64 < 0 {
				return nil, ErrNegativeIndex
			}
			k = int(i64)
		}
		return indexOf(host, k)
	}

	return nil, ErrIncompatibleType
}

func (ev *evaluator) evalArgs(ctx *Node, exprs []Expr) ([]interface{}, error) {
	args := make([]interface{}, 0, len(exprs))
	for _, a := range exprs {
		v, err := ev.eval(ctx, a)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}
	return args, nil
}

// childNamed returns the first child of n whose name matches.
func childNamed(n *Node, name string) (*Node, bool) {
	if n == nil {
		return nil, false
	}
	for _, c := range n.Out {
		if nameString(c.Data) == name {
			return c, true
		}
	}
	return nil, false
}

// builtinOf computes a built-in member of a node (Expr RFC 8.4).
//
// Built-ins are resolved last, after children and host members, so a document
// containing a node named _len keeps its own meaning.
func builtinOf(n *Node, name string) (interface{}, bool) {
	if n == nil {
		return nil, false
	}
	switch name {
	case "_this":
		return n, true
	case "_len":
		return int64(len(n.Out)), true
	case "_string":
		return n.String(), true
	case "_name", "_thisString":
		return n.ThisString(), true
	case "_raw":
		// What a substitution would write, vouched for as HTML by the
		// template that asks for it (Template RFC 4.1).
		return HTML(render(n)), true
	}
	return nil, false
}

// resolveMember applies one name to a value, in the order of Expr RFC 8.2.
// Built-ins are applied only when the evaluator allows them, which excludes
// strict path resolution (Expr RFC 8.4).
func (ev *evaluator) resolveMember(v interface{}, name string) (interface{}, bool) {
	if n, isNode := v.(*Node); isNode {
		if child, ok := childNamed(n, name); ok {
			return child, true
		}
		if m, ok := memberOf(scalarOf(n), name); ok {
			return m, true
		}
		if !ev.builtins {
			return nil, false
		}
		return builtinOf(n, name)
	}
	return memberOf(v, name)
}
