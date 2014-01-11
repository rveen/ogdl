// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bytes"
	"unicode"
)

// IsTextChar returns true for all integers > 32 and
// are not OGDL separators (parenthesis and comma)
func IsTextChar(c int) bool {
	if c > 32 && c != '(' && c != ')' && c != ',' {
		return true
	}
	return false
}

// IsEndChar returns true for all integers < 32 that are not newline, 
// carriage return or tab.
func IsEndChar(c int) bool {
	if c < 32 && c != '\t' && c != '\n' && c != '\r' {
		return true
	}
	return false
}

// IsSpaceChar returns true for space and tab
func IsSpaceChar(c int) bool {
	if c == 32 || c==9 {
		return true
	}
	return false
}

// IsTemplateTextChar returns true for all not END chars and not $
func IsTemplateTextChar(c int) bool {
	if !IsEndChar(c) && c != '$' {
		return true
	}
	return false
}

// IsOperatorChar returns true for all operator characters used in OGDL 
// expressions (those parsed by NewExpression).
func IsOperatorChar(c int) bool {
	if c < 0 {
		return false
	}
	if bytes.IndexByte([]byte("+-*/%&|!<>=~^"), byte(c)) != -1 {
		return true
	}
	return false
}

// ---- Following functions are the only ones that depend on Unicode --------

// IsLetter returns true if the given character is a letter, as per Unicode.
func IsLetter(c int) bool {
	return unicode.IsLetter(rune(c))
}

// IsDigit returns true if the given character a numeric digit, as per Unicode.
func IsDigit(c int) bool {
	return unicode.IsDigit(rune(c))
}

// isTokentChar returns true for letters, digits and _ (as per Unicode).
func IsTokenChar(c int) bool {
	return unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c == '_'
}
