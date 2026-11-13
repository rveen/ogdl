package ogdl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// Errors reported for the ill-formed inputs listed in RFC section 11.
var (
	ErrMixedIndentation        = errors.New("mixed indentation (SPACE and TAB) in one prefix")
	ErrInconsistentIndentation = errors.New("indentation character differs from the one used by the stream")
	ErrUnopenedDedent          = errors.New("dedent to an indentation level that was never opened")
	ErrUnterminatedQuoted      = errors.New("unterminated quoted string")
	ErrInvalidEscape           = errors.New("undefined escape sequence")
	ErrControlCharacter        = errors.New("control character other than TAB, LF and CR")
	ErrInvalidEncoding         = errors.New("invalid UTF-8 encoding")
	ErrBlockIndentation        = errors.New("block base indentation must be greater than that of the introducing line")
	ErrBlockNotLast            = errors.New("block introducer must be the last element on its line")
	ErrNameTooLong             = errors.New("name exceeds the implementation limit")
	ErrTooDeep                 = errors.New("nesting exceeds the implementation limit")
)

// SyntaxError locates an error in the stream. RFC section 11 requires the line
// and, where meaningful, the offset within the line.
type SyntaxError struct {
	Line int
	Col  int
	Err  error
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("ogdl: line %d, col %d: %v", e.Line, e.Col, e.Err)
}

func (e *SyntaxError) Unwrap() error { return e.Err }

// DefaultMaxNameLength is the documented limit required by RFC section 9.2 on
// the length in characters of a single word, quoted string or block.
const DefaultMaxNameLength = 1 << 20

type Token struct {
	Type  TokenType
	Value string
	Line  int
	Col   int
}

type TokenType int

const (
	// TokenIndent is the leading char_space run of a line. It is emitted only
	// at the start of a line, so the parser never has to guess whether spaces
	// are indentation (RFC 6.1).
	TokenIndent TokenType = iota
	// TokenSpace separates elements within a line.
	TokenSpace
	TokenWord
	TokenQuoted
	// TokenBlock is a word consisting of exactly one backslash: the block
	// introducer of RFC section 7, which is not a name.
	TokenBlock
	TokenComment
	TokenBreak
	TokenEOF
)

func (t Token) StringType() string {
	switch t.Type {
	case TokenIndent:
		return "Indent"
	case TokenSpace:
		return "Space"
	case TokenWord:
		return "Word"
	case TokenQuoted:
		return "Quoted"
	case TokenBlock:
		return "Block"
	case TokenComment:
		return "Comment"
	case TokenBreak:
		return "Break"
	case TokenEOF:
		return "EOF"
	}
	return ""
}

type Lexer struct {
	r       *bufio.Reader
	pending []rune // pushed-back input, consumed before r

	atLineStart   bool
	atStreamStart bool
	prevWasSpace  bool

	line int // 1-based line of the next character
	col  int // 1-based column of the next character

	// MaxNameLength bounds words, quoted strings and blocks (RFC 9.2, 12).
	MaxNameLength int
}

func isControl(r rune) bool {
	return (r >= 0x0000 && r <= 0x001F) || (r >= 0x007F && r <= 0x009F)
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func isBreak(r rune) bool {
	return r == '\n' || r == '\r'
}

// isCharEnd reports whether r is a control character other than TAB, LF and CR.
// Such characters MUST NOT appear in a stream (RFC 3.2).
func isCharEnd(r rune) bool {
	return isControl(r) && !isSpace(r) && !isBreak(r)
}

func isCharWord(r rune) bool {
	return !isSpace(r) && !isBreak(r) && !isCharEnd(r)
}

func NewLexer(rd io.Reader) *Lexer {
	return &Lexer{
		r:             bufio.NewReader(rd),
		atLineStart:   true,
		atStreamStart: true,
		line:          1,
		col:           1,
		MaxNameLength: DefaultMaxNameLength,
	}
}

func (lx *Lexer) errf(err error) error {
	return &SyntaxError{Line: lx.line, Col: lx.col, Err: err}
}

func (lx *Lexer) advance(r rune) {
	if r == '\n' {
		lx.line++
		lx.col = 1
		return
	}
	lx.col++
}

func (lx *Lexer) retreat(r rune) {
	if r == '\n' {
		lx.line--
		return
	}
	if lx.col > 1 {
		lx.col--
	}
}

// readRune returns the next character. Control characters other than TAB, LF
// and CR, and invalid encodings, are rejected here so that no other part of the
// lexer has to repeat the check (RFC 3.1, 3.2).
func (lx *Lexer) readRune() (rune, error) {
	var r rune
	if len(lx.pending) > 0 {
		r = lx.pending[0]
		lx.pending = lx.pending[1:]
	} else {
		var size int
		var err error
		r, size, err = lx.r.ReadRune()
		if err != nil {
			return 0, err
		}
		if r == utf8.RuneError && size == 1 {
			return 0, lx.errf(ErrInvalidEncoding)
		}
	}
	if isCharEnd(r) {
		return 0, lx.errf(ErrControlCharacter)
	}
	lx.advance(r)
	return r, nil
}

func (lx *Lexer) unread(r rune) {
	lx.pending = append([]rune{r}, lx.pending...)
	lx.retreat(r)
}

// pushBack returns already-read text to the input. Used by ReadBlock for the
// line that terminates a block.
func (lx *Lexer) pushBack(s string) {
	if s == "" {
		return
	}
	lx.pending = append([]rune(s), lx.pending...)
	lx.line -= strings.Count(s, "\n")
	lx.col = 1
}

// NextToken returns the next token. A TokenEOF is returned at end of stream and
// repeatedly thereafter.
func (lx *Lexer) NextToken() (Token, error) {
	line, col := lx.line, lx.col

	r, err := lx.readRune()
	if err == io.EOF {
		return Token{Type: TokenEOF, Line: line, Col: col}, nil
	}
	if err != nil {
		return Token{}, err
	}

	// A byte order mark is ignored, but only as the first character (RFC 3.4).
	if r == '\uFEFF' && lx.atStreamStart {
		lx.atStreamStart = false
		return lx.NextToken()
	}
	lx.atStreamStart = false

	// Break: LF, CR LF and CR are equivalent (RFC 4.1).
	if isBreak(r) {
		if r == '\r' {
			next, err := lx.readRune()
			if err == nil && next != '\n' {
				lx.unread(next)
			}
		}
		lx.atLineStart = true
		lx.prevWasSpace = false
		return Token{Type: TokenBreak, Line: line, Col: col}, nil
	}

	if isSpace(r) {
		buf := []rune{r}
		for {
			r2, err := lx.readRune()
			if err == io.EOF {
				break
			}
			if err != nil {
				return Token{}, err
			}
			if !isSpace(r2) {
				lx.unread(r2)
				break
			}
			buf = append(buf, r2)
		}
		typ := TokenSpace
		if lx.atLineStart {
			typ = TokenIndent
		}
		lx.atLineStart = false
		lx.prevWasSpace = true
		return Token{Type: typ, Value: string(buf), Line: line, Col: col}, nil
	}

	// A '#' introduces a comment only in the positions of RFC 4.6: at the start
	// of a line or after a space, and followed by a space, a break or EOS.
	if r == '#' && (lx.atLineStart || lx.prevWasSpace) {
		next, err := lx.readRune()
		switch {
		case err == io.EOF:
			lx.atLineStart = false
			lx.prevWasSpace = false
			return Token{Type: TokenComment, Line: line, Col: col}, nil
		case err != nil:
			return Token{}, err
		case isSpace(next) || isBreak(next):
			lx.unread(next)
			return lx.lexComment(line, col)
		default:
			lx.unread(next)
			return lx.lexWord('#', line, col)
		}
	}

	if r == '"' || r == '\'' {
		return lx.lexQuoted(r, line, col)
	}

	return lx.lexWord(r, line, col)
}

// lexWord reads a word (RFC 4.3).
func (lx *Lexer) lexWord(first rune, line, col int) (Token, error) {
	buf := []rune{first}
	for {
		r, err := lx.readRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Token{}, err
		}
		if !isCharWord(r) {
			lx.unread(r)
			break
		}
		buf = append(buf, r)
		if len(buf) > lx.MaxNameLength {
			return Token{}, lx.errf(ErrNameTooLong)
		}
	}
	lx.atLineStart = false
	lx.prevWasSpace = false

	// A word of exactly one backslash introduces a block and is not a name
	// (RFC 4.3, 7).
	if len(buf) == 1 && buf[0] == '\\' {
		return Token{Type: TokenBlock, Line: line, Col: col}, nil
	}
	return Token{Type: TokenWord, Value: string(buf), Line: line, Col: col}, nil
}

// lexQuoted reads a quoted string. Only the delimiter in use and the backslash
// are special inside it (RFC 3.2, 4.4).
func (lx *Lexer) lexQuoted(quote rune, line, col int) (Token, error) {
	buf := []rune{}
	for {
		r, err := lx.readRune()
		if err == io.EOF {
			return Token{}, &SyntaxError{Line: line, Col: col, Err: ErrUnterminatedQuoted}
		}
		if err != nil {
			return Token{}, err
		}

		if r == quote {
			break
		}

		if isBreak(r) {
			// A break contributes one LF; the indentation of the continuation
			// line is removed (RFC 4.4.2).
			if r == '\r' {
				next, err := lx.readRune()
				if err == nil && next != '\n' {
					lx.unread(next)
				}
			}
			buf = append(buf, '\n')
			for {
				next, err := lx.readRune()
				if err == io.EOF {
					return Token{}, &SyntaxError{Line: line, Col: col, Err: ErrUnterminatedQuoted}
				}
				if err != nil {
					return Token{}, err
				}
				if !isSpace(next) {
					lx.unread(next)
					break
				}
			}
			continue
		}

		if r == '\\' {
			next, err := lx.readRune()
			if err == io.EOF {
				return Token{}, &SyntaxError{Line: line, Col: col, Err: ErrUnterminatedQuoted}
			}
			if err != nil {
				return Token{}, err
			}
			switch next {
			case '\\', '\'', '"':
				buf = append(buf, next)
			case 'n':
				buf = append(buf, '\n')
			case 't':
				buf = append(buf, '\t')
			default:
				// Undefined escapes are reserved, not literal (RFC 4.4.1).
				return Token{}, lx.errf(ErrInvalidEscape)
			}
			continue
		}

		buf = append(buf, r)
		if len(buf) > lx.MaxNameLength {
			return Token{}, lx.errf(ErrNameTooLong)
		}
	}
	lx.atLineStart = false
	lx.prevWasSpace = false
	return Token{Type: TokenQuoted, Value: string(buf), Line: line, Col: col}, nil
}

// lexComment consumes to the end of the line. The terminating break is left in
// the input, so the parser still sees the line end.
func (lx *Lexer) lexComment(line, col int) (Token, error) {
	buf := []rune{}
	for {
		r, err := lx.readRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Token{}, err
		}
		if isBreak(r) {
			lx.unread(r)
			break
		}
		buf = append(buf, r)
	}
	lx.atLineStart = false
	lx.prevWasSpace = false
	return Token{Type: TokenComment, Value: strings.TrimPrefix(string(buf), " "), Line: line, Col: col}, nil
}

// physLine is one physical line as read by readPhysicalLine.
type physLine struct {
	prefix string // leading char_space run
	rest   string // the remainder, without the break
	raw    string // prefix + rest + break, for pushBack
	line   int
	eos    bool // no break: the stream ended here
}

func (p physLine) blank() bool { return p.rest == "" }

func (lx *Lexer) readPhysicalLine() (physLine, error) {
	pl := physLine{line: lx.line}
	var raw, prefix, rest strings.Builder
	inPrefix := true

	for {
		r, err := lx.readRune()
		if err == io.EOF {
			pl.eos = true
			break
		}
		if err != nil {
			return pl, err
		}
		if isBreak(r) {
			raw.WriteRune(r)
			if r == '\r' {
				next, err := lx.readRune()
				if err == nil {
					if next == '\n' {
						raw.WriteRune(next)
					} else {
						lx.unread(next)
					}
				}
			}
			break
		}
		raw.WriteRune(r)
		if inPrefix && isSpace(r) {
			prefix.WriteRune(r)
			continue
		}
		inPrefix = false
		rest.WriteRune(r)
	}

	pl.prefix = prefix.String()
	pl.rest = rest.String()
	pl.raw = raw.String()
	return pl, nil
}

// ReadBlock reads the block introduced on a line of indentation depth
// parentIndent, per RFC section 7. The introducing line's break must already
// have been consumed.
//
// check validates an indentation prefix against the stream's indentation
// character; it is not called for blank lines, whose whitespace is not an
// indentation prefix (RFC 4.2).
func (lx *Lexer) ReadBlock(parentIndent int, check func(prefix string, line int) error) (string, error) {
	base := -1
	var lines []string
	pendingBlanks := 0
	var blankRaw strings.Builder // the pending blank lines as read, for pushBack

	for {
		pl, err := lx.readPhysicalLine()
		if err != nil {
			return "", err
		}

		if pl.blank() {
			if pl.eos {
				break
			}
			// Blank lines never terminate a block. They are kept only if a
			// further block line follows (RFC 7).
			if base >= 0 {
				pendingBlanks++
				blankRaw.WriteString(pl.raw)
			}
			continue
		}

		depth := len([]rune(pl.prefix))

		if base < 0 {
			if depth <= parentIndent {
				lx.pushBack(pl.raw)
				lx.atLineStart = true
				return "", &SyntaxError{Line: pl.line, Col: 1, Err: ErrBlockIndentation}
			}
			if err := check(pl.prefix, pl.line); err != nil {
				return "", err
			}
			base = depth
		} else if depth < base {
			// Trailing blank lines are not part of the block; they are returned
			// to the input with the terminating line, so that a document
			// keeps them (RFC 7, Appendix B).
			lx.pushBack(blankRaw.String() + pl.raw)
			lx.atLineStart = true
			return strings.Join(lines, "\n"), nil
		} else if err := check(pl.prefix, pl.line); err != nil {
			return "", err
		}

		for i := 0; i < pendingBlanks; i++ {
			lines = append(lines, "")
		}
		pendingBlanks = 0
		blankRaw.Reset()

		// Only `base` indentation characters are removed; deeper indentation is
		// content (RFC 7).
		lines = append(lines, string([]rune(pl.prefix)[base:])+pl.rest)

		if pl.eos {
			break
		}
	}

	lx.atLineStart = true
	s := strings.Join(lines, "\n")
	if len(s) > lx.MaxNameLength {
		return "", lx.errf(ErrNameTooLong)
	}
	return s, nil
}
