package ogdl

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// This file implements the lexis and grammar of the expression RFC, sections 3
// and 4. Evaluation lives in eval.go.

// Errors reported when an expression cannot be parsed (Expr RFC 9).
var (
	ErrInvalidExpr     = errors.New("invalid expression")
	ErrMissingOperand  = errors.New("missing operand")
	ErrMissingParen    = errors.New("missing closing parenthesis")
	ErrReservedOp      = errors.New("operator reserved for assignment")
	ErrNumberRange     = errors.New("number out of range")
	ErrUnterminatedStr = errors.New("unterminated string literal")
)

// ExprError locates an error within an expression string.
type ExprError struct {
	Expr string
	Pos  int
	Err  error
}

func (e *ExprError) Error() string {
	return fmt.Sprintf("ogdl: expression %q, offset %d: %v", e.Expr, e.Pos, e.Err)
}

func (e *ExprError) Unwrap() error { return e.Err }

// Expr is a parsed expression. The concrete types below are the whole language:
// a literal, a path, or an operator applied to expressions.
type Expr interface {
	exprNode()
}

// Literal is a number, string or boolean written in the expression.
type Literal struct {
	Val interface{} // int64, float64, string or bool
}

// PathExpr addresses part of the tree, possibly calling into host values.
type PathExpr struct {
	Elems []pathElem
}

// Unary is "!" or "-" applied to an expression.
type Unary struct {
	Op string
	X  Expr
}

// Binary is an arithmetic, comparison or boolean operator.
type Binary struct {
	Op   string
	L, R Expr
}

func (*Literal) exprNode()  {}
func (*PathExpr) exprNode() {}
func (*Unary) exprNode()    {}
func (*Binary) exprNode()   {}

// precedence returns the binding power of a binary operator, tightest first
// (Expr RFC 4.1). It is Go's precedence without the bitwise operators. Level 0
// is reserved for assignment and is not parsed here.
func precedence(op string) int {
	switch op {
	case "*", "/", "%":
		return 5
	case "+", "-":
		return 4
	case "==", "!=", ">=", "<=", ">", "<":
		return 3
	case "&&":
		return 2
	case "||":
		return 1
	case "=", "+=", "-=", "*=", "/=", "%=":
		return 0
	}
	return -1
}

// isAssign reports whether op writes into the tree. Assignment is the only
// right-associative operator (Expr RFC 4.1).
func isAssign(op string) bool {
	switch op {
	case "=", "+=", "-=", "*=", "/=", "%=":
		return true
	}
	return false
}

// ParseExpr parses an expression.
func ParseExpr(s string) (Expr, error) {
	r := []rune(s)
	i := skipExprSpace(r, 0)
	if i == len(r) {
		return nil, &ExprError{Expr: s, Pos: 0, Err: ErrMissingOperand}
	}
	e, next, err := parseExprAt(s, r, i)
	if err != nil {
		return nil, err
	}
	next = skipExprSpace(r, next)
	if next != len(r) {
		return nil, &ExprError{Expr: s, Pos: next, Err: ErrInvalidExpr}
	}
	return e, nil
}

func skipExprSpace(r []rune, i int) int {
	for i < len(r) && (r[i] == ' ' || r[i] == '\t') {
		i++
	}
	return i
}

// parseExprAt parses one expression beginning at r[i], returning the offset
// just past it. It stops at any character it cannot use, which is what lets it
// be called for an argument list or an index and stop at ',' or ']'.
func parseExprAt(s string, r []rune, i int) (Expr, int, error) {
	return parseBinary(s, r, i, 0)
}

// parseBinary is precedence climbing: parse a unary expression, then absorb
// operators of at least the given precedence, left-associatively.
func parseBinary(s string, r []rune, i, minPrec int) (Expr, int, error) {
	left, i, err := parseUnary(s, r, i)
	if err != nil {
		return nil, 0, err
	}

	for {
		j := skipExprSpace(r, i)
		op, next, err := lexOperator(s, r, j)
		if err != nil {
			return nil, 0, err
		}
		prec := precedence(op)
		if op == "" || prec < minPrec {
			return left, i, nil
		}

		// Left-associative operators parse their right side at the next
		// precedence up; assignment, being right-associative, parses it at its
		// own level so that "a = b = 1" nests to the right (Expr RFC 4.1).
		nextPrec := prec + 1
		if isAssign(op) {
			nextPrec = prec
		}

		k := skipExprSpace(r, next)
		right, k, err := parseBinary(s, r, k, nextPrec)
		if err != nil {
			return nil, 0, err
		}
		left = &Binary{Op: op, L: left, R: right}
		i = k
	}
}

func parseUnary(s string, r []rune, i int) (Expr, int, error) {
	i = skipExprSpace(r, i)
	if i == len(r) {
		return nil, 0, &ExprError{Expr: s, Pos: i, Err: ErrMissingOperand}
	}

	// "!" and "-" are unary here; "!=" is binary and must not be mistaken for
	// unary "!" (Expr RFC 3.3).
	if r[i] == '!' && !(i+1 < len(r) && r[i+1] == '=') {
		x, next, err := parseUnary(s, r, i+1)
		if err != nil {
			return nil, 0, err
		}
		return &Unary{Op: "!", X: x}, next, nil
	}
	if r[i] == '-' {
		x, next, err := parseUnary(s, r, i+1)
		if err != nil {
			return nil, 0, err
		}
		return &Unary{Op: "-", X: x}, next, nil
	}

	return parsePrimary(s, r, i)
}

func parsePrimary(s string, r []rune, i int) (Expr, int, error) {
	i = skipExprSpace(r, i)
	if i == len(r) {
		return nil, 0, &ExprError{Expr: s, Pos: i, Err: ErrMissingOperand}
	}

	switch {
	case r[i] == '(':
		e, next, err := parseExprAt(s, r, i+1)
		if err != nil {
			return nil, 0, err
		}
		next = skipExprSpace(r, next)
		if next == len(r) || r[next] != ')' {
			return nil, 0, &ExprError{Expr: s, Pos: i, Err: ErrMissingParen}
		}
		return e, next + 1, nil

	case r[i] == '\'' || r[i] == '"':
		str, next, err := lexString(s, r, i)
		if err != nil {
			return nil, 0, err
		}
		return &Literal{Val: str}, next, nil

	case isDigit(r[i]):
		return lexNumber(s, r, i)

	case isTokenRune(r[i]):
		// "true" and "false" are literals, never paths (Expr RFC 3.4).
		start := i
		j := i
		for j < len(r) && isTokenRune(r[j]) {
			j++
		}
		word := string(r[start:j])
		if word == "true" || word == "false" {
			// Only when the word stands alone: "true.x" is not a literal.
			if j == len(r) || (r[j] != '.' && r[j] != '[' && r[j] != '{' && r[j] != '(') {
				return &Literal{Val: word == "true"}, j, nil
			}
		}
		elems, next, err := parsePathAt(s, r, i, true)
		if err != nil {
			return nil, 0, err
		}
		return &PathExpr{Elems: elems}, next, nil
	}

	return nil, 0, &ExprError{Expr: s, Pos: i, Err: ErrInvalidExpr}
}

// lexOperator reads a binary operator, longest match first. It returns an empty
// string when the next characters are not one.
func lexOperator(s string, r []rune, i int) (string, int, error) {
	if i >= len(r) {
		return "", i, nil
	}

	if i+1 < len(r) {
		two := string(r[i : i+2])
		switch two {
		case "==", "!=", ">=", "<=", "&&", "||", "+=", "-=", "*=", "/=", "%=":
			return two, i + 2, nil
		}
	}

	switch r[i] {
	case '+', '-', '*', '/', '%', '>', '<', '=':
		return string(r[i]), i + 1, nil
	case '&', '|':
		// A lone & or | is held for a possible bitwise layer (Expr RFC 3.3, 11).
		return "", 0, &ExprError{Expr: s, Pos: i, Err: ErrReservedOp}
	}
	return "", i, nil
}

// lexString reads a string literal, with the escapes of path RFC 4.3.
func lexString(s string, r []rune, i int) (string, int, error) {
	quote := r[i]
	open := i
	i++
	var sb strings.Builder
	for i < len(r) {
		c := r[i]
		if c == quote {
			return sb.String(), i + 1, nil
		}
		if isControl(c) {
			return "", 0, &ExprError{Expr: s, Pos: i, Err: ErrUnterminatedStr}
		}
		if c == '\\' {
			if i+1 == len(r) {
				return "", 0, &ExprError{Expr: s, Pos: open, Err: ErrUnterminatedStr}
			}
			switch r[i+1] {
			case '\\', '\'', '"':
				sb.WriteRune(r[i+1])
			default:
				return "", 0, &ExprError{Expr: s, Pos: i, Err: ErrInvalidEscape}
			}
			i += 2
			continue
		}
		sb.WriteRune(c)
		i++
	}
	return "", 0, &ExprError{Expr: s, Pos: open, Err: ErrUnterminatedStr}
}

// lexNumber reads an integer or float literal (Expr RFC 3.1). A sign is never
// part of the literal; it is the unary minus operator.
func lexNumber(s string, r []rune, i int) (Expr, int, error) {
	start := i
	for i < len(r) && isDigit(r[i]) {
		i++
	}
	isFloat := false
	if i < len(r) && r[i] == '.' && i+1 < len(r) && isDigit(r[i+1]) {
		isFloat = true
		i++
		for i < len(r) && isDigit(r[i]) {
			i++
		}
	}
	if i < len(r) && (r[i] == 'e' || r[i] == 'E') {
		j := i + 1
		if j < len(r) && (r[j] == '+' || r[j] == '-') {
			j++
		}
		if j < len(r) && isDigit(r[j]) {
			isFloat = true
			i = j
			for i < len(r) && isDigit(r[i]) {
				i++
			}
		}
	}

	text := string(r[start:i])
	if isFloat {
		f, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, 0, &ExprError{Expr: s, Pos: start, Err: ErrNumberRange}
		}
		return &Literal{Val: f}, i, nil
	}
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return nil, 0, &ExprError{Expr: s, Pos: start, Err: ErrNumberRange}
	}
	return &Literal{Val: n}, i, nil
}

// String renders an expression in a canonical, fully parenthesized form. It is
// a debugging aid and is not part of the language.
func exprString(e Expr) string {
	switch x := e.(type) {
	case *Literal:
		switch v := x.Val.(type) {
		case string:
			return "'" + v + "'"
		default:
			return toString(v)
		}
	case *PathExpr:
		return pathElemsString(x.Elems)
	case *Unary:
		return "(" + x.Op + exprString(x.X) + ")"
	case *Binary:
		return "(" + exprString(x.L) + x.Op + exprString(x.R) + ")"
	}
	return "?"
}
