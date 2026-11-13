package ogdl

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Errors reported when a path cannot be parsed (Path RFC 8).
var (
	ErrEmptyPath      = errors.New("empty path")
	ErrInvalidPath    = errors.New("invalid path")
	ErrUnterminated   = errors.New("unterminated quoted path element")
	ErrDanglingSelect = errors.New("selector must immediately follow a name element")
	ErrIndexTooLarge  = errors.New("index exceeds the implementation limit")
)

// maxPathIndex is the documented limit on an index or selector ordinal
// (Path RFC 9).
const maxPathIndex = 1 << 31

// PathError locates an error within a path string.
type PathError struct {
	Path string
	Pos  int
	Err  error
}

func (e *PathError) Error() string {
	return fmt.Sprintf("ogdl: path %q, offset %d: %v", e.Path, e.Pos, e.Err)
}

func (e *PathError) Unwrap() error { return e.Err }

type pathKind int

const (
	pathToken pathKind = iota
	pathIndex
	pathSelector
	// The two kinds below exist only in an expression context
	// (Expr RFC 5.2, 5.3).
	pathCall
	pathGroup
)

// pathElem is one element of a parsed path.
type pathElem struct {
	kind pathKind
	name string // pathToken: the text to match
	n    int    // pathSelector, and pathIndex in a strict path: the ordinal
	all  bool   // pathSelector: true for the empty selector {}
	expr Expr   // pathIndex in an expression, and pathGroup
	args []Expr // pathCall
}

// isTokenRune reports whether r may appear in an unquoted path token: a letter,
// a digit or an underscore, and nothing else (Path RFC 4.2).
//
// The class is deliberately narrow and defined by inclusion. Every other
// character stays free for the expression layer, so that `a-b` cannot become a
// single token today and a subtraction tomorrow. Names outside this class are
// addressed by quoting: host.'192.168.0.10'.
func isTokenRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isDigit reports whether r is an ASCII decimal digit. Other Unicode digits are
// deliberately excluded, so parsing does not vary with the Unicode version
// (Path RFC 4.1).
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// parsePath parses a strict OGDL path: the language of the path RFC, in which
// '(' and ')' are errors.
//
//	path    = element ( '.' element )*
//	element = ( token | quoted | index | selector ) ( index | selector )?
//	index   = '[' number ']'
//	selector = '{' number? '}'
//
// The dot is required between two tokens and optional before an index or a
// selector.
func parsePath(s string) ([]pathElem, error) {
	if s == "" {
		return nil, &PathError{Path: s, Pos: 0, Err: ErrEmptyPath}
	}
	r := []rune(s)
	elems, i, err := parsePathAt(s, r, 0, false)
	if err != nil {
		return nil, err
	}
	if i != len(r) {
		return nil, &PathError{Path: s, Pos: i, Err: ErrInvalidPath}
	}
	return elems, nil
}

// parsePathAt parses a path beginning at r[i] and returns the offset just past
// it.
//
// When extended is true it parses the path language of the expression RFC
// section 5 — calls, groups and expression indexes — and stops at the first
// character that cannot continue a path, leaving it to the caller. That is what
// lets an expression such as "a.b + 1" find the end of "a.b". When extended is
// false any such character is an error, so the strict language of the path RFC
// is unchanged.
func parsePathAt(s string, r []rune, i int, extended bool) ([]pathElem, int, error) {
	var elems []pathElem
	begin := true

	fail := func(pos int, err error) ([]pathElem, int, error) {
		return nil, 0, &PathError{Path: s, Pos: pos, Err: err}
	}

	for i < len(r) {
		if !begin {
			switch r[i] {
			case '.':
				// A dot before '(' introduces a group, not a name element.
				if extended && i+1 < len(r) && r[i+1] == '(' {
					e, next, err := parseGroup(s, r, i+2)
					if err != nil {
						return nil, 0, err
					}
					elems = append(elems, e)
					i = next
					continue
				}
				i++
				if i == len(r) {
					return fail(i, ErrInvalidPath)
				}
			case '[', '{':
				// The dot is optional before an index or a selector.
			case '(':
				if !extended {
					return fail(i, ErrInvalidPath)
				}
				args, next, err := parseArgs(s, r, i+1)
				if err != nil {
					return nil, 0, err
				}
				elems = append(elems, pathElem{kind: pathCall, args: args})
				i = next
				continue
			default:
				if extended {
					// Not part of the path: hand the character back.
					return elems, i, nil
				}
				return fail(i, ErrInvalidPath)
			}
		}
		begin = false

		switch {
		case r[i] == '\'' || r[i] == '"':
			quote := r[i]
			open := i
			i++
			var sb strings.Builder
			closed := false
			for i < len(r) {
				c := r[i]
				if c == quote {
					closed = true
					i++
					break
				}
				if isControl(c) {
					return fail(i, ErrUnterminated)
				}
				if c == '\\' {
					if i+1 == len(r) {
						return fail(open, ErrUnterminated)
					}
					switch r[i+1] {
					case '\\', '\'', '"':
						sb.WriteRune(r[i+1])
					default:
						return fail(i, ErrInvalidEscape)
					}
					i += 2
					continue
				}
				sb.WriteRune(c)
				i++
			}
			if !closed {
				return fail(open, ErrUnterminated)
			}
			elems = append(elems, pathElem{kind: pathToken, name: sb.String()})

		case r[i] == '[':
			start := i
			i++
			if extended {
				// An index may hold a whole expression (Expr RFC 5.1).
				e, next, err := parseExprAt(s, r, i)
				if err != nil {
					return nil, 0, err
				}
				next = skipExprSpace(r, next)
				if next == len(r) || r[next] != ']' {
					return fail(start, ErrInvalidPath)
				}
				elems = append(elems, pathElem{kind: pathIndex, expr: e})
				i = next + 1
				break
			}
			n, ok := 0, false
			for i < len(r) && isDigit(r[i]) {
				n = n*10 + int(r[i]-'0')
				ok = true
				if n > maxPathIndex {
					return fail(start, ErrIndexTooLarge)
				}
				i++
			}
			if !ok || i == len(r) || r[i] != ']' {
				return fail(start, ErrInvalidPath)
			}
			i++
			elems = append(elems, pathElem{kind: pathIndex, n: n})

		case r[i] == '{':
			start := i
			i++
			n, ok := 0, false
			for i < len(r) && isDigit(r[i]) {
				n = n*10 + int(r[i]-'0')
				ok = true
				if n > maxPathIndex {
					return fail(start, ErrIndexTooLarge)
				}
				i++
			}
			if i == len(r) || r[i] != '}' {
				return fail(start, ErrInvalidPath)
			}
			i++
			// A selector counts occurrences of a name, so it MUST immediately
			// follow a name element (Path RFC 5).
			if len(elems) == 0 || elems[len(elems)-1].kind != pathToken {
				return fail(start, ErrDanglingSelect)
			}
			elems = append(elems, pathElem{kind: pathSelector, n: n, all: !ok})

		case isTokenRune(r[i]):
			start := i
			for i < len(r) && isTokenRune(r[i]) {
				i++
			}
			elems = append(elems, pathElem{kind: pathToken, name: string(r[start:i])})

		default:
			return fail(i, ErrInvalidPath)
		}
	}

	if len(elems) == 0 {
		return fail(i, ErrEmptyPath)
	}
	return elems, i, nil
}

// parseGroup parses the body of a group element, ".(expr)", from just past the
// opening parenthesis (Expr RFC 5.3).
func parseGroup(s string, r []rune, i int) (pathElem, int, error) {
	e, next, err := parseExprAt(s, r, i)
	if err != nil {
		return pathElem{}, 0, err
	}
	next = skipExprSpace(r, next)
	if next == len(r) || r[next] != ')' {
		return pathElem{}, 0, &PathError{Path: s, Pos: next, Err: ErrInvalidPath}
	}
	return pathElem{kind: pathGroup, expr: e}, next + 1, nil
}

// parseArgs parses an argument list from just past the opening parenthesis
// (Expr RFC 5.2). An empty list is allowed; a trailing comma is not.
func parseArgs(s string, r []rune, i int) ([]Expr, int, error) {
	var args []Expr
	i = skipExprSpace(r, i)
	if i < len(r) && r[i] == ')' {
		return args, i + 1, nil
	}
	for {
		e, next, err := parseExprAt(s, r, i)
		if err != nil {
			return nil, 0, err
		}
		args = append(args, e)
		i = skipExprSpace(r, next)
		if i == len(r) {
			return nil, 0, &PathError{Path: s, Pos: i, Err: ErrInvalidPath}
		}
		switch r[i] {
		case ')':
			return args, i + 1, nil
		case ',':
			i = skipExprSpace(r, i+1)
			if i < len(r) && r[i] == ')' {
				// Trailing comma.
				return nil, 0, &PathError{Path: s, Pos: i, Err: ErrInvalidPath}
			}
		default:
			return nil, 0, &PathError{Path: s, Pos: i, Err: ErrInvalidPath}
		}
	}
}

// Get evaluates an OGDL path against the receiver and returns the node it
// selects. It never returns nil: a path that matches nothing, or that cannot be
// parsed, yields an empty node, so calls may be chained.
//
//	g.Get("eth0.ip").String()      // the text under eth0.ip
//	g.Get("list[2]").ThisString()  // the third child of list
//	g.Get("chapter{1}.title")      // the title of the second chapter
//	g.Get("chapter.p{}")           // a set node holding every p
//
// A token or a numbered selector selects a node, so String() reads the text
// below it and ThisString() its own text. The empty selector {} selects a set:
// a synthetic node whose children are the matches, iterated through Out.
func (n *Node) Get(path string) *Node {
	r, err := n.GetErr(path)
	if err != nil {
		return &Node{}
	}
	return r
}

// GetErr is Get, reporting a malformed path as a *PathError. A well-formed path
// that matches nothing is not an error; it returns an empty node.
func (n *Node) GetErr(path string) (*Node, error) {
	elems, err := parsePath(path)
	if err != nil {
		return &Node{}, err
	}
	return resolvePath(n, elems), nil
}

// Path returns the parsed form of a path as a readable string. It is a
// debugging aid, and reports malformed paths as an error.
func Path(s string) (string, error) {
	elems, err := parsePath(s)
	if err != nil {
		return "", err
	}
	return pathElemsString(elems), nil
}

func pathElemsString(elems []pathElem) string {
	var parts []string
	for _, e := range elems {
		switch e.kind {
		case pathToken:
			parts = append(parts, fmt.Sprintf("token(%s)", e.name))
		case pathIndex:
			if e.expr != nil {
				parts = append(parts, fmt.Sprintf("index(%s)", exprString(e.expr)))
			} else {
				parts = append(parts, fmt.Sprintf("index(%d)", e.n))
			}
		case pathSelector:
			if e.all {
				parts = append(parts, "select(all)")
			} else {
				parts = append(parts, fmt.Sprintf("select(%d)", e.n))
			}
		case pathCall:
			var args []string
			for _, a := range e.args {
				args = append(args, exprString(a))
			}
			parts = append(parts, fmt.Sprintf("call(%s)", strings.Join(args, ",")))
		case pathGroup:
			parts = append(parts, fmt.Sprintf("group(%s)", exprString(e.expr)))
		}
	}
	return strings.Join(parts, " ")
}
