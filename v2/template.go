package ogdl

import (
	"errors"
	"fmt"
	"html"
	"reflect"
	"strings"
	"unicode"
)

// This file implements the template language of the template RFC: text in which
// '$' switches to the expression language.

// Errors reported for a malformed or unprocessable template (Template RFC 6).
var (
	ErrUnexpectedEnd    = errors.New("$end with no open $if or $for")
	ErrUnexpectedElse   = errors.New("$else or $elseif with no open $if")
	ErrDoubleElse       = errors.New("$else after $else")
	ErrMissingEnd       = errors.New("missing $end")
	ErrBreakOutsideLoop = errors.New("$break outside a loop")
	ErrBadFor           = errors.New("$for needs a variable name and an expression")
	ErrNotIterable      = errors.New("value is not iterable")
	ErrUnterminatedTpl  = errors.New("unterminated template directive")
)

// TemplateError locates an error within a template.
type TemplateError struct {
	Pos int
	Err error
}

func (e *TemplateError) Error() string {
	return fmt.Sprintf("ogdl: template offset %d: %v", e.Pos, e.Err)
}

func (e *TemplateError) Unwrap() error { return e.Err }

// DefaultMaxTemplateDepth bounds nesting of conditionals and loops
// (Template RFC 7).
const DefaultMaxTemplateDepth = 100

// Template is a parsed template.
type Template struct {
	body []tmpl

	// MaxDepth bounds conditional and loop nesting.
	MaxDepth int

	// EscapeHTML makes every substitution HTML-escaped, except a value of type
	// HTML, which is written unchanged (Template RFC 4.1). Text outside
	// substitutions is never escaped.
	EscapeHTML bool
}

// HTML is text vouched for as safe HTML. A template that escapes HTML writes
// it unchanged; any other value it escapes.
//
// Only code produces one: the host, by creating the value, or a template, by
// the built-in _raw. A document carries no types, so nothing read from a
// stream is HTML by itself.
type HTML string

// tmpl is one piece of a template.
type tmpl interface{ tmplNode() }

type tmplText struct{ s string }

type tmplSubst struct {
	e   Expr
	pos int
}

type tmplSilent struct {
	e   Expr
	pos int
}

type tmplIf struct {
	conds  []Expr // one per if/elseif branch
	blocks [][]tmpl
	els    []tmpl // nil when there is no $else
	pos    int
}

type tmplFor struct {
	name string
	src  Expr
	body []tmpl
	pos  int
}

type tmplBreak struct{ pos int }

func (*tmplText) tmplNode()   {}
func (*tmplSubst) tmplNode()  {}
func (*tmplSilent) tmplNode() {}
func (*tmplIf) tmplNode()     {}
func (*tmplFor) tmplNode()    {}
func (*tmplBreak) tmplNode()  {}

// ParseTemplate parses a template.
func ParseTemplate(s string) (*Template, error) {
	p := &tmplParser{src: s, r: []rune(s)}
	body, err := p.parseBlock(false)
	if err != nil {
		return nil, err
	}
	if p.i < len(p.r) {
		// parseBlock stopped at a directive that closes nothing.
		return nil, &TemplateError{Pos: p.i, Err: p.stopErr}
	}
	return &Template{body: body, MaxDepth: DefaultMaxTemplateDepth}, nil
}

// ParseHTMLTemplate parses a template whose substitutions are HTML-escaped.
func ParseHTMLTemplate(s string) (*Template, error) {
	t, err := ParseTemplate(s)
	if err != nil {
		return nil, err
	}
	t.EscapeHTML = true
	return t, nil
}

type tmplParser struct {
	src     string
	r       []rune
	i       int
	stop    string // the directive that ended the current block
	stopPos int
	stopErr error
}

// parseBlock reads template pieces until the end of the input or a directive
// that closes the enclosing block. The closing directive is left in p.stop.
func (p *tmplParser) parseBlock(nested bool) ([]tmpl, error) {
	var out []tmpl
	var text strings.Builder

	flush := func() {
		if text.Len() > 0 {
			out = append(out, &tmplText{s: text.String()})
			text.Reset()
		}
	}

	for p.i < len(p.r) {
		c := p.r[p.i]
		if c != '$' {
			text.WriteRune(c)
			p.i++
			continue
		}

		start := p.i
		p.i++ // consume '$'

		if p.i == len(p.r) {
			// A trailing '$' is literal (Template RFC 2.6).
			text.WriteRune('$')
			break
		}

		switch n := p.r[p.i]; {
		case n == '$':
			text.WriteRune('$')
			p.i++
			continue

		case n == '(':
			p.i++
			e, next, err := parseExprAt(p.src, p.r, p.i)
			if err != nil {
				return nil, err
			}
			next = skipExprSpace(p.r, next)
			if next == len(p.r) || p.r[next] != ')' {
				return nil, &TemplateError{Pos: start, Err: ErrUnterminatedTpl}
			}
			p.i = next + 1
			flush()
			out = append(out, &tmplSilent{e: e, pos: start})
			continue

		case n == '{':
			p.i++
			p.i = skipExprSpace(p.r, p.i)
			elems, next, err := parsePathAt(p.src, p.r, p.i, true)
			if err != nil {
				return nil, err
			}
			next = skipExprSpace(p.r, next)
			if next == len(p.r) || p.r[next] != '}' {
				return nil, &TemplateError{Pos: start, Err: ErrUnterminatedTpl}
			}
			p.i = next + 1
			flush()
			out = append(out, &tmplSubst{e: &PathExpr{Elems: elems}, pos: start})
			continue

		case n == '_' || unicode.IsLetter(n):
			// A directive or a substitution.
			word := p.peekWord()
			switch word {
			case "if":
				flush()
				node, err := p.parseIf(start)
				if err != nil {
					return nil, err
				}
				out = append(out, node)
				continue

			case "for":
				flush()
				node, err := p.parseFor(start)
				if err != nil {
					return nil, err
				}
				out = append(out, node)
				continue

			case "break":
				p.i += len(word)
				flush()
				out = append(out, &tmplBreak{pos: start})
				continue

			case "end", "else", "elseif":
				if !nested {
					p.stopErr = ErrUnexpectedEnd
					if word != "end" {
						p.stopErr = ErrUnexpectedElse
					}
					p.i = start
					flush()
					return out, nil
				}
				p.i += len(word)
				p.stop = word
				p.stopPos = start
				flush()
				return out, nil
			}

			elems, next, err := parsePathAt(p.src, p.r, p.i, true)
			if err != nil {
				return nil, err
			}
			p.i = next
			flush()
			out = append(out, &tmplSubst{e: &PathExpr{Elems: elems}, pos: start})
			continue

		default:
			// A '$' before anything else is literal (Template RFC 2.6).
			text.WriteRune('$')
			continue
		}
	}

	flush()
	p.stop = ""
	return out, nil
}

// peekWord returns the run of token characters at the cursor.
func (p *tmplParser) peekWord() string {
	j := p.i
	for j < len(p.r) && isTokenRune(p.r[j]) {
		j++
	}
	return string(p.r[p.i:j])
}

// parseCond consumes "(expression)" after a directive name.
func (p *tmplParser) parseCond(name string, start int) (Expr, error) {
	p.i += len(name)
	p.i = skipExprSpace(p.r, p.i)
	if p.i == len(p.r) || p.r[p.i] != '(' {
		return nil, &TemplateError{Pos: start, Err: ErrUnterminatedTpl}
	}
	p.i++
	e, next, err := parseExprAt(p.src, p.r, p.i)
	if err != nil {
		return nil, err
	}
	next = skipExprSpace(p.r, next)
	if next == len(p.r) || p.r[next] != ')' {
		return nil, &TemplateError{Pos: start, Err: ErrUnterminatedTpl}
	}
	p.i = next + 1
	return e, nil
}

func (p *tmplParser) parseIf(start int) (tmpl, error) {
	node := &tmplIf{pos: start}

	cond, err := p.parseCond("if", start)
	if err != nil {
		return nil, err
	}

	for {
		block, err := p.parseBlock(true)
		if err != nil {
			return nil, err
		}
		node.conds = append(node.conds, cond)
		node.blocks = append(node.blocks, block)

		switch p.stop {
		case "end":
			return node, nil

		case "elseif":
			cond, err = p.parseCond("", p.stopPos)
			if err != nil {
				return nil, err
			}

		case "else":
			els, err := p.parseBlock(true)
			if err != nil {
				return nil, err
			}
			if p.stop == "else" || p.stop == "elseif" {
				return nil, &TemplateError{Pos: p.stopPos, Err: ErrDoubleElse}
			}
			if p.stop != "end" {
				return nil, &TemplateError{Pos: start, Err: ErrMissingEnd}
			}
			node.els = els
			return node, nil

		default:
			return nil, &TemplateError{Pos: start, Err: ErrMissingEnd}
		}
	}
}

func (p *tmplParser) parseFor(start int) (tmpl, error) {
	p.i += len("for")
	p.i = skipExprSpace(p.r, p.i)
	if p.i == len(p.r) || p.r[p.i] != '(' {
		return nil, &TemplateError{Pos: start, Err: ErrUnterminatedTpl}
	}
	p.i++
	p.i = skipExprSpace(p.r, p.i)

	// The loop variable is a name: a token beginning with a letter or an
	// underscore (Template RFC 2.5). Digits are path tokens, so "$for(1, b)"
	// would otherwise bind a variable named 1.
	name := p.peekWord()
	if name == "" {
		return nil, &TemplateError{Pos: start, Err: ErrBadFor}
	}
	if first := []rune(name)[0]; first != '_' && !unicode.IsLetter(first) {
		return nil, &TemplateError{Pos: start, Err: ErrBadFor}
	}
	p.i += len([]rune(name))
	p.i = skipExprSpace(p.r, p.i)
	if p.i == len(p.r) || p.r[p.i] != ',' {
		return nil, &TemplateError{Pos: start, Err: ErrBadFor}
	}
	p.i++

	src, next, err := parseExprAt(p.src, p.r, p.i)
	if err != nil {
		return nil, err
	}
	next = skipExprSpace(p.r, next)
	if next == len(p.r) || p.r[next] != ')' {
		return nil, &TemplateError{Pos: start, Err: ErrUnterminatedTpl}
	}
	p.i = next + 1

	body, err := p.parseBlock(true)
	if err != nil {
		return nil, err
	}
	if p.stop != "end" {
		return nil, &TemplateError{Pos: start, Err: ErrMissingEnd}
	}

	return &tmplFor{name: name, src: src, body: body, pos: start}, nil
}

// Process renders the template against a context node.
//
// Errors are reported, not swallowed: a path that matches nothing renders as
// nothing, but a malformed expression or a failed call is an error
// (Template RFC 6).
func (t *Template) Process(ctx *Node) (string, error) {
	var sb strings.Builder
	ev := &evaluator{maxDepth: DefaultMaxEvalDepth, builtins: true}
	st := &tmplState{ev: ev, maxDepth: t.MaxDepth, escapeHTML: t.EscapeHTML}
	if _, err := st.run(t.body, ctx, &sb); err != nil {
		return sb.String(), err
	}
	return sb.String(), nil
}

// ProcessString parses and processes a template in one step.
func ProcessString(tpl string, ctx *Node) (string, error) {
	t, err := ParseTemplate(tpl)
	if err != nil {
		return "", err
	}
	return t.Process(ctx)
}

// ProcessHTMLString parses and processes an HTML-escaping template in one step.
func ProcessHTMLString(tpl string, ctx *Node) (string, error) {
	t, err := ParseHTMLTemplate(tpl)
	if err != nil {
		return "", err
	}
	return t.Process(ctx)
}

type tmplState struct {
	ev         *evaluator
	depth      int
	maxDepth   int
	inLoop     int
	escapeHTML bool
}

// run processes a block. It reports whether a $break was hit.
func (st *tmplState) run(body []tmpl, ctx *Node, sb *strings.Builder) (bool, error) {
	for _, node := range body {
		switch n := node.(type) {

		case *tmplText:
			sb.WriteString(n.s)

		case *tmplSubst:
			v, err := st.ev.eval(ctx, n.e)
			if err != nil {
				return false, &TemplateError{Pos: n.pos, Err: err}
			}
			sb.WriteString(st.render(v))

		case *tmplSilent:
			if _, err := st.ev.eval(ctx, n.e); err != nil {
				return false, &TemplateError{Pos: n.pos, Err: err}
			}

		case *tmplIf:
			brk, err := st.runIf(n, ctx, sb)
			if err != nil || brk {
				return brk, err
			}

		case *tmplFor:
			brk, err := st.runFor(n, ctx, sb)
			if err != nil {
				return false, err
			}
			_ = brk // a break ends this loop only

		case *tmplBreak:
			if st.inLoop == 0 {
				return false, &TemplateError{Pos: n.pos, Err: ErrBreakOutsideLoop}
			}
			return true, nil
		}
	}
	return false, nil
}

func (st *tmplState) runIf(n *tmplIf, ctx *Node, sb *strings.Builder) (bool, error) {
	if err := st.enter(n.pos); err != nil {
		return false, err
	}
	defer func() { st.depth-- }()

	for i, cond := range n.conds {
		v, err := st.ev.eval(ctx, cond)
		if err != nil {
			return false, &TemplateError{Pos: n.pos, Err: err}
		}
		if toBool(v) {
			return st.run(n.blocks[i], ctx, sb)
		}
	}
	if n.els != nil {
		return st.run(n.els, ctx, sb)
	}
	return false, nil
}

func (st *tmplState) runFor(n *tmplFor, ctx *Node, sb *strings.Builder) (bool, error) {
	if err := st.enter(n.pos); err != nil {
		return false, err
	}
	defer func() { st.depth-- }()

	v, err := st.loopSource(ctx, n.src)
	if err != nil {
		return false, &TemplateError{Pos: n.pos, Err: err}
	}

	items, err := iterate(v)
	if err != nil {
		return false, &TemplateError{Pos: n.pos, Err: err}
	}
	if len(items) == 0 {
		return false, nil
	}

	// The variable is bound in the evaluator's scope, never in the tree, and
	// names the element itself (Template RFC 5.2). It hides a node of the same
	// name and any outer variable for the duration of the loop; the outer
	// binding is restored however the loop ends.
	ev := st.ev
	if ev.vars == nil {
		ev.vars = map[string]*Node{}
	}
	outer, hadOuter := ev.vars[n.name]
	defer func() {
		if hadOuter {
			ev.vars[n.name] = outer
		} else {
			delete(ev.vars, n.name)
		}
	}()

	st.inLoop++
	defer func() { st.inLoop-- }()

	for _, item := range items {
		ev.vars[n.name] = item
		brk, err := st.run(n.body, ctx, sb)
		if err != nil {
			return false, err
		}
		if brk {
			break
		}
	}
	return false, nil
}

// loopSource evaluates the expression of a $for (Template RFC 5.1).
//
// A path is taken before operand simplification: a node with one leaf child,
// or none, is still a node, and iterates its children. Simplifying it first
// would turn a one-element list into a string and an empty list into its own
// name, neither of which is iterable. The one exception is a single leaf child
// holding a host value, which is how a host value lives in the tree (Set): it
// is the node's value, and is iterated as such.
func (st *tmplState) loopSource(ctx *Node, e Expr) (interface{}, error) {
	p, ok := e.(*PathExpr)
	if !ok {
		return st.ev.eval(ctx, e)
	}
	v, err := st.ev.evalPath(ctx, p.Elems)
	if err != nil {
		return nil, err
	}
	n, ok := v.(*Node)
	if !ok || n == nil {
		return v, nil
	}
	switch s := scalarOf(n).(type) {
	case *Node, string, []byte:
		return n, nil
	default:
		return s, nil
	}
}

func (st *tmplState) enter(pos int) error {
	st.depth++
	if st.depth > st.maxDepth {
		return &TemplateError{Pos: pos, Err: ErrEvalTooDeep}
	}
	return nil
}

// iterate returns the elements a loop walks over (Template RFC 5.1).
func iterate(v interface{}) ([]*Node, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case *Node:
		if x == nil {
			return nil, nil
		}
		return x.Out, nil
	}

	// A host slice or array becomes one node per element. Its length is taken
	// from the value itself: a nil element is an element, not the end.
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, ErrNotIterable
	}
	out := make([]*Node, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out = append(out, &Node{Data: rv.Index(i).Interface()})
	}
	return out, nil
}

// render turns a value into template output, escaping it when the template
// escapes HTML (Template RFC 4.1).
func (st *tmplState) render(v interface{}) string {
	if !st.escapeHTML {
		return render(v)
	}
	if h, ok := scalarOf(v).(HTML); ok {
		return string(h)
	}
	return html.EscapeString(render(v))
}

// render turns a value into template output (Template RFC 4).
func render(v interface{}) string {
	s := scalarOf(v)
	if n, ok := s.(*Node); ok && n != nil {
		// A node that did not simplify renders as what lies below it.
		text, _ := n.Canonical()
		return strings.TrimSuffix(text, "\n")
	}
	return toString(s)
}
