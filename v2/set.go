package ogdl

import (
	"errors"
)

// This file implements assignment targets: resolving a path to the node an
// assignment writes to, creating what is missing, and performing the write
// (Expr RFC 6.5).

// Errors reported when a path cannot be an assignment target (Expr RFC 6.5, 9).
var (
	ErrNotAssignable = errors.New("assignment target is not a path")
	ErrTargetIsSet   = errors.New("assignment target is a set")
	ErrTargetIndex   = errors.New("index target is more than one past the end")
	ErrTargetMissing = errors.New("assignment target does not exist")
)

// Set assigns a value at a path, creating any missing name elements, and
// returns the node written to. It never returns nil: a malformed or
// unassignable path yields an empty node, so calls may be chained.
//
//	g.Set("a.b", "d")     // a -> b -> d, creating a and b if needed
//	g.Set("list[1]", 99)  // replaces the second child of list
//
// The value replaces every child of the target node, because that is how a
// document holds content: the node names it, the child holds it. A *Node value
// contributes its children, not itself.
func (n *Node) Set(path string, value interface{}) *Node {
	r, err := n.SetErr(path, value)
	if err != nil {
		return &Node{}
	}
	return r
}

// SetErr is Set, reporting why an assignment could not be made.
func (n *Node) SetErr(path string, value interface{}) (*Node, error) {
	elems, err := parsePath(path)
	if err != nil {
		return &Node{}, err
	}
	ev := &evaluator{maxDepth: DefaultMaxEvalDepth}
	target, item, err := ev.target(n, elems)
	if err != nil {
		return &Node{}, err
	}
	assign(target, value, item)
	return target, nil
}

// target resolves the node an assignment writes to, creating missing name
// elements along the way (Expr RFC 6.5).
//
// Creation is undone if resolution later fails, so a rejected assignment leaves
// the tree exactly as it was rather than littering it with the prefix of a path
// that turned out to be unusable.
// item reports whether the value replaces the addressed node itself rather than
// its children: that is what an index addresses (Expr RFC 6.5).
func (ev *evaluator) target(root *Node, elems []pathElem) (n *Node, item bool, err error) {
	if root == nil {
		return nil, false, ErrNotAssignable
	}

	var created []*Node // parents that had a child appended, in order
	defer func() {
		if err == nil {
			return
		}
		for i := len(created) - 1; i >= 0; i-- {
			p := created[i]
			p.Out = p.Out[:len(p.Out)-1]
		}
	}()

	node := root
	name := ""     // the name matched by the last name element
	parent := root // the node that name was matched in

	isVar := false // node is a loop variable reached by the first element

	for i, e := range elems {
		item = e.kind == pathIndex
		switch e.kind {
		case pathToken, pathGroup:
			n := e.name
			if e.kind == pathGroup {
				v, err := ev.eval(root, e.expr)
				if err != nil {
					return nil, false, err
				}
				n = toString(scalarOf(v))
				if n == "" {
					return nil, false, ErrTargetMissing
				}
			}
			if i == 0 {
				// A loop variable is the element itself (Template RFC 5.2):
				// assigning through it writes into the element.
				if v, ok := ev.variable(n); ok {
					node, name, isVar = v, n, true
					continue
				}
			}
			isVar = false
			child, ok := childNamed(node, n)
			if !ok {
				// Missing name elements are created (Expr RFC 6.5).
				child = &Node{Data: n}
				node.Out = append(node.Out, child)
				created = append(created, node)
			}
			parent = node
			node = child
			name = n

		case pathIndex:
			k := e.n
			if e.expr != nil {
				v, err := ev.eval(root, e.expr)
				if err != nil {
					return nil, false, err
				}
				i64, ok := toInt64(v)
				if !ok {
					return nil, false, ErrNotANumber
				}
				if i64 < 0 {
					return nil, false, ErrNegativeIndex
				}
				k = int(i64)
			}
			switch {
			case k < len(node.Out):
				parent = node
				node = node.Out[k]
			case k == len(node.Out):
				// One past the end appends; further would leave holes.
				child := &Node{}
				node.Out = append(node.Out, child)
				created = append(created, node)
				parent = node
				node = child
			default:
				return nil, false, ErrTargetIndex
			}
			name = ""
			isVar = false

		case pathSelector:
			if e.all {
				return nil, false, ErrTargetIsSet
			}
			if name == "" {
				return nil, false, ErrNotAssignable
			}
			if isVar {
				// A variable is occurrence 0 of its name, and the only one.
				if e.n != 0 {
					return nil, false, ErrTargetMissing
				}
				continue
			}
			// A selector addresses an existing occurrence; it cannot create one.
			found := (*Node)(nil)
			k := 0
			for _, child := range parent.Out {
				if nameString(child.Data) != name {
					continue
				}
				if k == e.n {
					found = child
					break
				}
				k++
			}
			if found == nil {
				return nil, false, ErrTargetMissing
			}
			node = found

		case pathCall:
			return nil, false, ErrNotAssignable
		}
	}

	if node == root {
		// The path addressed the context itself, which is not a target.
		return nil, false, ErrNotAssignable
	}
	return node, item, nil
}

// assign writes a value into the target node (Expr RFC 6.5).
//
// A name element addresses a node that names its content, so the value replaces
// the node's children. An index addresses an item, so the value replaces the
// node itself. The asymmetry is the same one paths already have: a.b is read
// with String(), list[0] with ThisString().
func assign(target *Node, value interface{}, item bool) {
	v, isNode := value.(*Node)

	if item {
		if isNode && v != nil {
			target.Data = v.Data
			target.Out = append([]*Node(nil), v.Out...)
			return
		}
		target.Data = value
		target.Out = nil
		return
	}

	if isNode && v != nil {
		// A node contributes its children; its own content is discarded,
		// because the target already has a name.
		target.Out = append([]*Node(nil), v.Out...)
		return
	}
	target.Out = []*Node{{Data: value}}
}
