package ogdl

import (
	"io"
	"strings"
)

// DefaultMaxDepth is the documented nesting limit required by RFC section 9.2.
const DefaultMaxDepth = 1000

// frame is an open node: the head of a line, with the indentation depth of
// that line. Only heads are ever open (RFC 6.5).
type frame struct {
	node  *Node
	depth int
}

type Parser struct {
	lx *Lexer

	// indentChar is the stream's indentation character, determined by the first
	// line with a non-empty indentation prefix (RFC 6.3).
	indentChar rune

	stack []frame

	// doc, when set, receives the comments and layout of the stream
	// (ParseDocument); nil for a plain parse.
	doc     *Document
	pending []trivia // blank and comment-only lines not yet attached

	prevDepth int  // indentation depth of the last content-bearing line
	hasPrev   bool // whether any content-bearing line has been seen

	// MaxDepth bounds the nesting depth (RFC 9.2, 12).
	MaxDepth int

	// Chain reads a line as ogdl v1 did: its elements form a chain, each
	// the child of the one before (`a b c` is a → b → c), and a block after
	// them belongs to the last. Without it, the line is a head with its
	// arguments as children (`a b c` is a with b and c; RFC 6.5). Lines
	// indented below attach to the head either way. The v1-named helpers
	// (FromString, FromBytes, FromFile) set it.
	Chain bool
}

func NewParser(r io.Reader) (*Parser, error) {
	return &Parser{
		lx:       NewLexer(r),
		MaxDepth: DefaultMaxDepth,
	}, nil
}

// checkIndent enforces the indentation character rules of RFC 6.3 and 6.4. It
// must not be called for blank lines, whose whitespace is not an indentation
// prefix (RFC 4.2).
func (p *Parser) checkIndent(prefix string, line int) error {
	if prefix == "" {
		return nil
	}
	if strings.ContainsRune(prefix, ' ') && strings.ContainsRune(prefix, '\t') {
		return &SyntaxError{Line: line, Col: 1, Err: ErrMixedIndentation}
	}
	c := rune(prefix[0])
	if p.indentChar == 0 {
		p.indentChar = c
		return nil
	}
	if c != p.indentChar {
		return &SyntaxError{Line: line, Col: 1, Err: ErrInconsistentIndentation}
	}
	return nil
}

// isOpen reports whether depth n is the indentation depth of a line whose head
// is still open. Depth 0 is always open, since the root is the parent of every
// depth-zero head (RFC 6.7).
func (p *Parser) isOpen(n int) bool {
	if n == 0 {
		return true
	}
	for _, f := range p.stack {
		if f.depth == n {
			return true
		}
	}
	return false
}

// closeTo pops every open head whose line is not shallower than depth, and
// returns the node that a line at that depth attaches to.
func (p *Parser) closeTo(depth int) *Node {
	for len(p.stack) > 1 && p.stack[len(p.stack)-1].depth >= depth {
		p.stack = p.stack[:len(p.stack)-1]
	}
	return p.stack[len(p.stack)-1].node
}

// line holds the tokens of one line, already classified.
type line struct {
	indent   string
	elements []Token // words and quoted strings, each one node
	block    bool
	comment  bool
	text     string // the comment's text, without "#"
	eof      bool
	num      int
}

// blank reports whether the line has no content at all: no elements, no block
// and no comment (RFC 4.2).
func (l line) blank() bool {
	return len(l.elements) == 0 && !l.block && !l.comment
}

// content reports whether the line contributes nodes.
func (l line) content() bool {
	return len(l.elements) > 0 || l.block
}

// readLine assembles one line from the token stream.
func (p *Parser) readLine() (line, error) {
	l := line{num: p.lx.line}
	for {
		tok, err := p.lx.NextToken()
		if err != nil {
			return l, err
		}
		switch tok.Type {
		case TokenIndent:
			l.indent = tok.Value
		case TokenSpace:
			// Separator; carries no structure (RFC 6.1).
		case TokenWord, TokenQuoted:
			if l.block {
				return l, &SyntaxError{Line: tok.Line, Col: tok.Col, Err: ErrBlockNotLast}
			}
			l.elements = append(l.elements, tok)
		case TokenBlock:
			if l.block {
				return l, &SyntaxError{Line: tok.Line, Col: tok.Col, Err: ErrBlockNotLast}
			}
			l.block = true
		case TokenComment:
			l.comment = true
			l.text = tok.Value
		case TokenBreak:
			return l, nil
		case TokenEOF:
			l.eof = true
			return l, nil
		}
	}
}

// Parse reads the whole stream and returns the root node (RFC 2.3). The root is
// unnamed and is the parent of every depth-zero node.
func (p *Parser) Parse() (*Node, error) {
	root := &Node{}
	p.stack = []frame{{node: root, depth: -1}}
	p.prevDepth = 0
	p.hasPrev = false
	p.pending = nil

	for {
		l, err := p.readLine()
		if err != nil {
			return nil, err
		}

		if !l.blank() {
			if err := p.checkIndent(l.indent, l.num); err != nil {
				return nil, err
			}
		}

		depth := len([]rune(l.indent))

		if !l.content() {
			if p.doc != nil {
				p.collect(l, depth)
			}
			if l.eof {
				break
			}
			continue
		}

		// A dedent must land on a level that is open (RFC 6.7).
		if p.hasPrev && depth < p.prevDepth && !p.isOpen(depth) {
			return nil, &SyntaxError{Line: l.num, Col: 1, Err: ErrUnopenedDedent}
		}
		p.prevDepth = depth
		p.hasPrev = true

		if p.doc != nil {
			p.attachEnd(depth)
		}

		// The first element is the head of the line; the others, and a block,
		// are its children. Only the head stays open (RFC 6.5, 6.6).
		parent := p.closeTo(depth)
		var head, tail *Node
		if len(l.elements) > 0 {
			head = &Node{Data: l.elements[0].Value}
			parent.Out = append(parent.Out, head)
			if len(p.stack) > p.MaxDepth {
				e := l.elements[0]
				return nil, &SyntaxError{Line: e.Line, Col: e.Col, Err: ErrTooDeep}
			}
			p.stack = append(p.stack, frame{node: head, depth: depth})
			last := head
			for _, e := range l.elements[1:] {
				arg := &Node{Data: e.Value}
				if p.Chain {
					if len(p.stack)+(len(l.elements)-1) > p.MaxDepth {
						return nil, &SyntaxError{Line: e.Line, Col: e.Col, Err: ErrTooDeep}
					}
					last.Out = append(last.Out, arg)
					last = arg
				} else {
					head.Out = append(head.Out, arg)
				}
				p.recordInline(arg)
			}
			tail = last
		}

		if l.block {
			s, err := p.lx.ReadBlock(depth, p.checkIndent)
			if err != nil {
				return nil, err
			}
			b := &Node{Data: s}
			if head != nil && p.Chain {
				tail.Out = append(tail.Out, b)
				p.recordInline(b)
			} else if head != nil {
				head.Out = append(head.Out, b)
				p.recordInline(b)
			} else {
				// A bare block line is a leaf at the line's own depth.
				parent.Out = append(parent.Out, b)
				head = b
			}
		}

		if p.doc != nil {
			p.attachLine(head, l)
		}

		if l.eof {
			break
		}
	}

	if p.doc != nil {
		p.attachEOF(root)
	}
	return root, nil
}

// Parse reads an OGDL document and returns its root node.
func Parse(r io.Reader) (*Node, error) {
	p, err := NewParser(r)
	if err != nil {
		return nil, err
	}
	return p.Parse()
}

// ParseString reads an OGDL document from a string.
func ParseString(s string) (*Node, error) {
	return Parse(strings.NewReader(s))
}
