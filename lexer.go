// Adapted from golang.org/src/bufio/bufio.go, with this license:
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file associated with
// the Go sources.
//

// Objectives
//
// - reduce the number of Read's to the underlying reader.
// - multiple byte unreads possible.
// - methods for reading Byte, Rune, String, Token, Space, etc, adapted to the needs
//   of parsing OGDL text and paths
//

package ogdl

import (
	"bytes"
	"errors"
	"io"
	"unicode"
	"unicode/utf8"
)

const (
	bufSize  = 4096
	halfSize = 4096 / 2
)

var (
	errNegativeRead = errors.New("reader returned negative count from Read")
)

// Lexer implements buffering for an io.Reader object, with multiple byte unread
// operations allowed.
type Lexer struct {
	buf          []byte
	rd           io.Reader // reader provided by the client
	r            int       // buf read position
	lastByte     int       // If not -1, then the buffer contains the last byte of the stream at this position.
	lastRuneSize []int     // used by UnreadRune.
	err          error
}

const maxConsecutiveEmptyReads = 100

// NewLexer returns a new Lexer whose buffer has the default size.
func NewLexer(rd io.Reader) *Lexer {
	p := Lexer{}
	p.rd = rd
	p.lastByte = bufSize
	p.buf = make([]byte, bufSize)
	p.r = -1
	p.fill()
	return &p
}

// fill reads a new chunk into the buffer.
//
// The first time, the buffer is filled completely. After reading the last byte from
// the buffer, the last half is preserved and moved to the start, and the other
// half filled with new bytes, if available.
func (p *Lexer) fill() {

	if p.r >= 0 && p.r < bufSize {
		return
	}

	// Read new data: try a limited number of times.
	// The first time read the full buffer, else only half.
	offset := 0
	if p.r >= 0 {
		copy(p.buf, p.buf[halfSize:])
		p.r = halfSize
		offset = halfSize
	} else {
		p.r = 0
	}

	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err := p.rd.Read(p.buf[offset:])
		if n < 0 {
			panic(errNegativeRead)
		}
		if err != nil {
			p.err = err
			break
		}

		offset += n
		if offset >= halfSize {
			break
		}
	}

	p.lastByte = offset
}

func (p *Lexer) Error() error {
	err := p.err
	p.err = nil
	return err
}

// PeekByte returns the next byte witohut consuming it
func (p *Lexer) PeekByte() byte {
	c, err := p.Byte()

	if err != nil {
		return 0
	}

	p.UnreadByte()
	return c
}

// PeekRune returns the next rune witohut consuming it
func (p *Lexer) PeekRune() (rune, error) {
	r, err := p.Rune()

	if err != nil {
		return 0, nil
	}

	return r, p.UnreadRune()
}

// Byte reads and returns a single byte.
// If no byte is available, returns 0 and an error.
func (p *Lexer) Byte() (byte, error) {
	if p.lastByte < bufSize && p.r >= p.lastByte {
		p.r = p.lastByte + 1
		return 0, ErrEOS
	}

	c := p.buf[p.r]
	p.r++
	p.fill()
	return c, nil
}

// UnreadByte unreads the last byte. It can unread all buffered bytes.
func (p *Lexer) UnreadByte() {
	if p.r <= 0 {
		return
	}

	p.r--
	return
}

// Rune reads a single UTF-8 encoded Unicode character and returns the
// rune. If the encoded rune is invalid, it consumes one byte
// and returns unicode.ReplacementChar (U+FFFD) with a size of 1.
func (p *Lexer) Rune() (rune, error) {

	p.fill()

	r, size := rune(p.buf[p.r]), 1
	if r >= utf8.RuneSelf {
		r, size = utf8.DecodeRune(p.buf[p.r:p.lastByte])
	}
	p.r += size
	p.lastRuneSize = append(p.lastRuneSize, size)
	return r, nil
}

// UnreadRune unreads the last rune.
func (p *Lexer) UnreadRune() error {
	if len(p.lastRuneSize) == 0 {
		return ErrInvalidUnread
	}

	p.r -= p.lastRuneSize[len(p.lastRuneSize)-1]
	p.lastRuneSize = p.lastRuneSize[:len(p.lastRuneSize)-1]

	return nil
}

// String is a concatenation of characters that are > 0x20
func (p *Lexer) String() (string, bool) {

	var buf []byte

	for {
		c, _ := p.Byte()
		if !IsTextChar(c) {
			break
		}
		buf = append(buf, c)
	}

	p.UnreadByte()
	return string(buf), len(buf) > 0
}

// StringStop is a concatenation of text bytes that are not in the parameter stopBytes
func (p *Lexer) StringStop(stopBytes []byte) (string, bool) {

	var buf []byte

	for {
		c, _ := p.Byte()
		if !IsTextChar(c) || bytes.IndexByte(stopBytes, c) != -1 {
			break
		}
		buf = append(buf, c)
	}

	p.UnreadByte()
	return string(buf), len(buf) > 0
}

// Break (= newline) is NL, CR or CR+NL
func (p *Lexer) Break() bool {
	c, _ := p.Byte()

	if c == '\r' {
		c, _ = p.Byte()
	}

	if c == '\n' {
		return true
	}

	p.UnreadByte()
	return false
}

// End returns true if the end of stream has been reached.
func (p *Lexer) End() bool {
	c, err := p.Byte()

	if err != nil {
		return true
	}

	if IsEndChar(c) {
		return true
	}
	p.UnreadByte()
	return false
}

// WhiteSpace is equivalent to Space | Break. It consumes all white space,
// whether spaces, tabs or newlines
func (p *Lexer) WhiteSpace() bool {

	any := false
	for {
		c, _ := p.Byte()
		if c != 13 && c != 10 && c != 9 && c != 32 {
			break
		}
		any = true
	}

	p.UnreadByte()
	return any
}

// Space is (0x20|0x09)+. It returns a boolean indicating
// if space has been found, and an integer indicating
// how many spaces, iff uniform (either all 0x20 or 0x09)
func (p *Lexer) Space() (int, byte) {

	// Need a bufio that unreads many bytes for the Block case

	n := 0
	m := 0

	for {
		c, _ := p.Byte()
		if c != '\t' && c != ' ' {
			break
		}
		if c == ' ' {
			n++
		} else {
			m++
		}

	}

	p.UnreadByte()

	var r byte
	if m == 0 {
		r = ' '
	} else if n == 0 {
		r = '\t'
	}

	return n + m, r
}

// Quoted string. Can have newlines in it. It returns the string if any, a bool
// indicating if a quoted string was found, and a possible error.
func (p *Lexer) Quoted(ind int) (string, bool, error) {

	c1, _ := p.Byte()
	if c1 != '"' && c1 != '\'' {
		p.UnreadByte()
		return "", false, nil
	}

	var buf []byte
	var c2 byte

	for {
		c, _ := p.Byte()
		if IsEndChar(c) {
			return "", false, ErrUnterminatedQuotedString
		}

		if c == c1 && c2 != '\\' {
			break
		}
		if c == '\\' {
			c2 = c
			continue
		}

		// \" -> "
		// \' -> '
		if c2 == '\\' && !(c != '\'' || c == '"') {
			buf = append(buf, '\\')
		}

		buf = append(buf, c)

		if c == 10 {
			n, u := p.Space()
			if u == 0 {
				return "", false, ErrSpaceNotUniform
			}
			// There are n spaces. Skip lnl spaces and add rest.
			for ; n-ind > 0; n-- {
				buf = append(buf, u)
			}
		}
		c2 = c
	}

	return string(buf), true, nil
}

// Token8 reads from the Parser input stream and returns
// a token, if any. A token is defined as a sequence of
// Unicode letters and/or numbers and/or _.
func (p *Lexer) Token8() (string, bool) {

	var buf []rune

	for {
		c, _ := p.Rune()
		if !isTokenChar(c) {
			break
		}
		buf = append(buf, c)
	}

	p.UnreadRune()
	return string(buf), len(buf) > 0
}

// Integer returns true if it finds an (unsigned) integer at the current
// parser position. It returns also the number found.
func (p *Lexer) Integer() (string, bool) {

	var buf []byte

	for {
		c, _ := p.Byte()
		if !IsDigit(rune(c)) {
			break
		}
		buf = append(buf, c)
	}

	p.UnreadByte()
	return string(buf), len(buf) > 0
}

// Number returns true if it finds a number at the current
// parser position. It returns also the number found.
// TODO recognize exp notation ?
func (p *Lexer) Number() (string, bool) {

	var buf []byte
	var sign byte
	point := false

	c := p.PeekByte()
	if c == '-' || c == '+' {
		sign = c
		p.Byte()
	}

	for {
		c, _ := p.Byte()
		if !IsDigit(rune(c)) {
			if !point && c == '.' {
				point = true
				buf = append(buf, c)
				continue
			}
			break
		}
		buf = append(buf, c)
	}

	p.UnreadByte()
	if sign == '-' {
		return "-" + string(buf), len(buf) > 0
	}
	return string(buf), len(buf) > 0
}

// Operator returns true if it finds an operator at the current parser position
// It returns also the operator found.
func (p *Lexer) Operator() (string, bool) {

	var buf []byte

	for {
		c, _ := p.Byte()
		if !isOperatorChar(c) {
			break
		}
		buf = append(buf, c)
	}

	p.UnreadByte()
	return string(buf), len(buf) > 0
}

// TemplateText parses text in a template.
func (p *Lexer) TemplateText() (string, bool) {
	var buf []byte

	for {
		c, _ := p.Byte()
		if !isTemplateTextChar(c) {
			break
		}
		buf = append(buf, c)
	}

	p.UnreadByte()
	return string(buf), len(buf) > 0
}

// Comment consumes anything from # up to the end of the line.
func (p *Lexer) Comment() bool {
	c, _ := p.Byte()

	if c == '#' {
		c, _ = p.Byte()
		if IsSpaceChar(c) {
			for {
				c, _ = p.Byte()
				if IsEndChar(c) || IsBreakChar(c) {
					break
				}
			}
			return true
		}
		p.UnreadByte()
	}
	p.UnreadByte()
	return false
}

// Block ::= '\\' NL LINES_OF_TEXT
func (p *Lexer) Block(nsp int) (string, bool) {

	c, _ := p.Byte()
	if c != '\\' {
		p.UnreadByte()
		return "", false
	}

	c, _ = p.Byte()
	if c != 10 && c != 13 {
		p.UnreadByte()
		p.UnreadByte()
		return "", false
	}
	// Read NL if CR was found
	if c == 13 {
		p.Byte()
	}

	buffer := &bytes.Buffer{}

	// read lines while indentation is > nsp

	for {
		ns, u := p.Space()

		if u == 0 || ns <= nsp {
			break
		}

		// Adjust indentation if less that initial

		// Read bytes until end of line
		for {
			c, _ = p.Byte()

			if IsEndChar(c) {
				break
			}

			if c == 13 {
				continue
			}

			buffer.WriteByte(c)

			if c == 10 {
				break
			}
		}
	}

	// Remove trailing NLs
	s := buffer.String()
	for {
		if len(s) == 0 {
			break
		}

		c := s[len(s)-1]
		if c == 10 {
			s = s[0 : len(s)-1]
		} else {
			break
		}
	}

	return s, true
}

// Scalar ::= quoted | string
func (p *Lexer) Scalar(n int) (string, bool) {
	b, ok, _ := p.Quoted(n)
	if ok {
		return b, true
	}
	return p.String()
}

// IsTextChar returns true for all integers > 32 and
// are not OGDL separators (parenthesis and comma)
func IsTextChar(c byte) bool {
	return c > 32
}

// IsEndChar returns true for all integers < 32 that are not newline,
// carriage return or tab.
func IsEndChar(c byte) bool {
	return c < 32 && c != '\t' && c != '\n' && c != '\r'
}

// IsEndRune returns true for all integers < 32 that are not newline,
// carriage return or tab.
func IsEndRune(c rune) bool {
	return c < 32 && c != '\t' && c != '\n' && c != '\r'
}

// IsBreakChar returns true for 10 and 13 (newline and carriage return)
func IsBreakChar(c byte) bool {
	return c == 10 || c == 13
}

// IsSpaceChar returns true for space and tab
func IsSpaceChar(c byte) bool {
	return c == 32 || c == 9
}

// IsTemplateTextChar returns true for all not END chars and not $
func isTemplateTextChar(c byte) bool {
	return !IsEndChar(c) && c != '$'
}

// IsOperatorChar returns true for all operator characters used in OGDL
// expressions (those parsed by NewExpression).
func isOperatorChar(c byte) bool {
	if c < 0 {
		return false
	}
	return bytes.IndexByte([]byte("+-*/%&|!<>=~^"), c) != -1
}

// ---- Following functions are the only ones that depend on Unicode --------

// IsLetter returns true if the given character is a letter, as per Unicode.
func IsLetter(c rune) bool {
	return unicode.IsLetter(c) || c == '_'
}

// IsDigit returns true if the given character a numeric digit, as per Unicode.
func IsDigit(c rune) bool {
	return unicode.IsDigit(c)
}

// IsTokenChar returns true for letters, digits and _ (as per Unicode).
func isTokenChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_'
}
