package ogdl

import (
	"bytes"
	"strings"
)

func (g *Graph) Html(path string) string {
	if g == nil {
		return ""
	}

	buffer := &bytes.Buffer{}

	// Do not print the 'root' node
	for _, node := range g.Out {
		node._html(0, buffer, false, path)
	}

	// remove trailing \n

	s := buffer.String()

	if len(s) == 0 {
		return ""
	}

	if s[len(s)-1] == '\n' {
		s = s[0 : len(s)-1]
	}

	// unquote

	if s[0] == '"' {
		s = s[1 : len(s)-1]
		// But then also replace \"
		s = strings.Replace(s, "\\\"", "\"", -1)
	}

	return s
}

// _text is the private, lower level, implementation of Text().
// It takes two parameters, the level and a buffer to which the
// result is printed.
func (g *Graph) _html(n int, buffer *bytes.Buffer, show bool, path string) {

	if g == nil {
		return
	}

	sp := ""
	for i := 0; i < n; i++ {
		sp += "&nbsp;&nbsp;"
	}

	/*
	   When printing strings with newlines, there are two possibilities:
	   block or quoted. Block is cleaner, but limited to leaf nodes. If the node
	   is not leaf (it has subnodes), then we are forced to print a multiline
	   quoted string.

	   If the string has no newlines but spaces or special characters, then the
	   same rule applies: quote those nodes that are non-leaf, print a block
	   otherways.

	   [!] Cannot print blocks at level 0? Or can we?
	*/

	s := _string(g.This)
	if s == "" {
		s = "_"
	}

	// If this is a leaf node, do not add a link
	if g.Len() > 0 {
		buffer.WriteString("<a href='" + path + "/" + s + "'>")
	}

	if strings.ContainsAny(s, "\n\r \t'\",()") {

		// print quoted, but not at level 0
		// Do not convert " to \" below if level==0 !
		if n > 0 {
			buffer.WriteString(sp) /* [:len(sp)-1]) */
			buffer.WriteByte('"')
		}

		var c, cp byte

		cp = 0

		for i := 0; i < len(s); i++ {
			c = s[i] // byte, not rune
			if c == 13 {
				continue // ignore CR's
			} else if c == 10 {
				buffer.WriteByte('\n')
				buffer.WriteString(sp)
			} else if c == '"' && n > 0 {
				if cp != '\\' {
					buffer.WriteString("\\\"")
				}
			} else {
				buffer.WriteByte(c)
			}
			cp = c
		}

		if n > 0 {
			buffer.WriteString("\"")
		}
		buffer.WriteString("<br>\n")
	} else {
		if len(s) == 0 && !show {
			n--
		} else {
			if len(s) == 0 && show {
				s = "_"
			}
			buffer.WriteString(sp)
			buffer.WriteString(s)
			if g.Len() > 0 {
				buffer.WriteString("</a>")
			}
			buffer.WriteString("<br>\n")
		}
	}

	for _, node := range g.Out {
		node._html(n+1, buffer, show, path+"/"+s)
	}
}
