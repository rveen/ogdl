package ogdl

import (
	"io"
	"strings"
	"unicode/utf8"
)

// Document is a parsed stream together with the presentation that the data
// model discards (RFC 2.5): comments, blank lines, and which nodes were written
// inline. It exists so that a program can read a hand-maintained file, change
// the tree, and write it back without losing the author's comments.
//
// Presentation is keyed by node identity. A node the program keeps keeps its
// comments; a node it removes or replaces takes them along; a node it creates
// has none, and is laid out by the compact rule (Node.Compact).
type Document struct {
	Root *Node

	// Comments holds the comments attached to each node (RFC Appendix B).
	// Nodes without comments have no entry. The root's After holds the
	// comments at the end of the stream.
	Comments map[*Node]*Comments

	// inline records, for each parsed node, whether it was written inline (an
	// argument or a block on its parent's line) or on a line of its
	// own. Nodes created later have no entry.
	inline map[*Node]bool
}

// Comments are the comment and blank lines that belong to a node.
//
// Each entry of Before and After is either a comment line, written with its
// leading "#", or the empty string for a blank line. Line is the comment that
// ends the node's own line, or empty.
type Comments struct {
	Before []string // lines above the node
	Line   string   // trailing comment on the node's line
	After  []string // lines below the node's last child, indented as its children
}

func (c *Comments) empty() bool {
	return c == nil || len(c.Before) == 0 && c.Line == "" && len(c.After) == 0
}

// ParseDocument parses a stream like Parse, and also retains its comments and
// layout.
func ParseDocument(r io.Reader) (*Document, error) {
	p, err := NewParser(r)
	if err != nil {
		return nil, err
	}
	d := &Document{Comments: map[*Node]*Comments{}, inline: map[*Node]bool{}}
	p.doc = d
	root, err := p.Parse()
	if err != nil {
		return nil, err
	}
	d.Root = root
	return d, nil
}

// ParseDocumentString parses a document from a string.
func ParseDocumentString(s string) (*Document, error) {
	return ParseDocument(strings.NewReader(s))
}

// comments returns the entry for n, creating it.
func (d *Document) comments(n *Node) *Comments {
	c := d.Comments[n]
	if c == nil {
		c = &Comments{}
		d.Comments[n] = c
	}
	return c
}

// Format serializes the document with its comments. Layout follows the
// compact form (Node.Compact), except that a parsed node stays inline or on its
// own line as it was written. Indentation is normalized to two spaces, and
// blocks and multi-line strings become quoted strings.
func (d *Document) Format() (string, error) {
	if d.Root == nil {
		return "", nil
	}
	pr := printer{doc: d}
	pr.root(d.Root)
	return pr.sb.String(), pr.err
}

// Text is Format without the error. A node that cannot be serialized is
// omitted, as in Node.Text.
func (d *Document) Text() string {
	s, _ := d.Format()
	return s
}

// Compact returns the compact serialization of the receiver's children: each
// node is written as a head followed by the run of its leading children that
// are leaves, written as arguments. The run is inlined if it has one element or the
// line fits in 80 columns; the remaining children are indented below.
//
// The compact form is not canonical — equality is defined by Canonical — but
// it is deterministic and reparses to the same tree.
func (n *Node) Compact() (string, error) {
	pr := printer{}
	pr.root(n)
	return pr.sb.String(), pr.err
}

// compactWidth is the line width up to which a run of several children is
// written inline.
const compactWidth = 80

// trivia is a blank or comment-only line awaiting attachment to a node.
type trivia struct {
	blank bool
	text  string // the comment line, with its "#"
	depth int
}

// collect queues a blank or comment-only line. Blank lines before the first
// content are dropped and consecutive ones are collapsed.
func (p *Parser) collect(l line, depth int) {
	if l.blank() {
		n := len(p.pending)
		if n == 0 && !p.hasPrev || n > 0 && p.pending[n-1].blank {
			return
		}
		p.pending = append(p.pending, trivia{blank: true})
		return
	}
	if l.comment {
		p.pending = append(p.pending, trivia{text: commentText(l.text), depth: depth})
	}
}

func commentText(s string) string {
	if s == "" {
		return "#"
	}
	return "# " + s
}

// attachEnd gives the queued comments that are indented deeper than a line
// about to start at depth to the open heads that line closes, as After
// comments. A comment goes to the deepest such head shallower than itself;
// targets never get deeper, so order is preserved. The first comment that has
// no target, and everything after it, stays queued for the next node.
func (p *Parser) attachEnd(depth int) {
	limit := len(p.stack)
	blank := -1 // index of a queued blank line preceding the current comment
	for i := 0; i < len(p.pending); i++ {
		t := p.pending[i]
		if t.blank {
			blank = i
			continue
		}
		j := limit - 1
		for j >= 1 && p.stack[j].depth >= t.depth {
			j--
		}
		if j < 1 || p.stack[j].depth < depth {
			if blank >= 0 {
				i = blank
			}
			p.pending = p.pending[i:]
			return
		}
		c := p.doc.comments(p.stack[j].node)
		if blank >= 0 {
			c.After = append(c.After, "")
		}
		c.After = append(c.After, t.text)
		blank = -1
		limit = j + 1
	}
	if blank >= 0 {
		p.pending = p.pending[blank:]
	} else {
		p.pending = nil
	}
}

// attachLine gives the remaining queued lines, and the line's trailing comment,
// to the node that heads the line.
func (p *Parser) attachLine(head *Node, l line) {
	p.doc.inline[head] = false
	if len(p.pending) == 0 && !l.comment {
		return
	}
	c := p.doc.comments(head)
	for _, t := range p.pending {
		if t.blank {
			c.Before = append(c.Before, "")
		} else {
			c.Before = append(c.Before, t.text)
		}
	}
	if l.comment {
		c.Line = commentText(l.text)
	}
	p.pending = nil
}

// attachEOF attaches the comments at the end of the stream: those indented
// under an open head to that head, the rest to the root. Trailing blank lines
// are dropped.
func (p *Parser) attachEOF(root *Node) {
	p.attachEnd(0)
	for len(p.pending) > 0 && p.pending[len(p.pending)-1].blank {
		p.pending = p.pending[:len(p.pending)-1]
	}
	if len(p.pending) == 0 {
		return
	}
	c := p.doc.comments(root)
	for _, t := range p.pending {
		if t.blank {
			c.After = append(c.After, "")
		} else {
			c.After = append(c.After, t.text)
		}
	}
	p.pending = nil
}

// recordInline notes that n was written inline on its parent's line.
func (p *Parser) recordInline(n *Node) {
	if p.doc != nil {
		p.doc.inline[n] = true
	}
}

// printer writes the compact form, and with a document, its comments and
// recorded layout.
type printer struct {
	sb  strings.Builder
	doc *Document
	err error
}

func (pr *printer) fail(err error) {
	if pr.err == nil {
		pr.err = err
	}
}

func (pr *printer) comments(n *Node) *Comments {
	if pr.doc == nil {
		return nil
	}
	return pr.doc.Comments[n]
}

func (pr *printer) root(r *Node) {
	for _, n := range r.Out {
		pr.node(n, 0)
	}
	if c := pr.comments(r); c != nil {
		pr.trivia(c.After, 0)
	}
}

func (pr *printer) node(n *Node, level int) {
	c := pr.comments(n)
	if c != nil {
		pr.trivia(c.Before, level)
	}
	name, err := formatName(n.Data)
	if err != nil {
		pr.fail(err)
		return
	}
	line := strings.Repeat("  ", level) + name
	texts := pr.inlineRun(n, utf8.RuneCountInString(line))
	for _, t := range texts {
		line += " " + t
	}
	if c != nil && c.Line != "" {
		if s, ok := pr.commentLines(strings.ReplaceAll(c.Line, "\n", " ")); ok {
			line += " " + s[0]
		}
	}
	pr.sb.WriteString(line)
	pr.sb.WriteByte('\n')
	for _, m := range n.Out[len(texts):] {
		pr.node(m, level+1)
	}
	if c != nil {
		pr.trivia(c.After, level+1)
	}
}

// inlineRun returns the inline text of the children written on n's line.
func (pr *printer) inlineRun(n *Node, width int) []string {
	var texts []string
	for _, m := range n.Out {
		t, ok := pr.inlineText(m)
		if !ok {
			break
		}
		texts = append(texts, t)
	}
	def := len(texts)
	if def > 1 {
		w := width
		for _, t := range texts {
			w += 1 + utf8.RuneCountInString(t)
		}
		if w > compactWidth {
			def = 0
		}
	}
	if pr.doc == nil {
		return texts[:def]
	}
	k := 0
	for ; k < len(texts); k++ {
		in, known := pr.doc.inline[n.Out[k]]
		if known && !in || !known && k >= def {
			break
		}
	}
	return texts[:k]
}

// inlineText returns how m is written on its parent's line, as an argument,
// if it can be: only a leaf without comments is inline.
func (pr *printer) inlineText(m *Node) (string, bool) {
	if len(m.Out) > 0 || !pr.comments(m).empty() {
		return "", false
	}
	s, err := formatName(m.Data)
	return s, err == nil
}

// trivia writes comment and blank lines at the given level.
func (pr *printer) trivia(lines []string, level int) {
	for _, s := range lines {
		if s == "" {
			pr.sb.WriteByte('\n')
			continue
		}
		cl, ok := pr.commentLines(s)
		if !ok {
			continue
		}
		for _, l := range cl {
			pr.sb.WriteString(strings.Repeat("  ", level))
			pr.sb.WriteString(l)
			pr.sb.WriteByte('\n')
		}
	}
}

// commentLines turns comment text into well-formed comment lines: one per
// line of text, each starting with "#" and a space, so that it lexes as a
// comment and not as a word (RFC 4.6).
func (pr *printer) commentLines(s string) ([]string, bool) {
	for _, r := range s {
		if isCharEnd(r) {
			pr.fail(ErrControlCharacter)
			return nil, false
		}
	}
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(s, "\r", "\n"), "\n") {
		l = strings.TrimRight(l, " \t")
		switch {
		case l == "#":
		case strings.HasPrefix(l, "# "), strings.HasPrefix(l, "#\t"):
		case strings.HasPrefix(l, "#"):
			l = "# " + l[1:]
		default:
			l = commentText(l)
		}
		out = append(out, l)
	}
	return out, true
}
