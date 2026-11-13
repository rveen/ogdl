package ogdl

import (
	"fmt"
	"strings"
)

// Node is a node of an OGDL tree: one item of data and an ordered list of
// children (RFC 2.1, 2.2).
//
// Data holds any Go value. It is deliberately not called a name: what the RFC
// calls a node's name is the *textual* form this data takes in a stream, which
// is only one of the things Data can hold. Parsing always produces string data,
// since a stream carries nothing but text; other types arise when a tree is
// built programmatically or by a later evaluation stage.
//
// Comparison and serialization use the textual form, via nameString.
type Node struct {
	Data interface{}
	Out  []*Node
}

// nameString resolves a node's data to the text that represents it in a stream:
// the OGDL name of the node.
func nameString(v interface{}) string {
	switch s := v.(type) {
	case nil:
		return ""
	case string:
		return s
	case []byte:
		return string(s)
	case *Node:
		return s.String()
	}
	return fmt.Sprint(v)
}

func (n *Node) N(i int) *Node {
	if i < 0 || i >= len(n.Out) {
		return &Node{}
	}
	return n.Out[i]
}

// S returns the first child whose textual form equals s, or an empty node.
func (n *Node) S(s string) *Node {
	for _, nn := range n.Out {
		if s == nameString(nn.Data) {
			return nn
		}
	}
	return &Node{}
}

// ThisString returns the textual form of this node's own data.
func (n *Node) ThisString() string {
	if n == nil {
		return ""
	}
	return nameString(n.Data)
}

// String returns the textual form of this node's first child.
func (n *Node) String() string {
	if n == nil || len(n.Out) == 0 {
		return ""
	}
	return nameString(n.Out[0].Data)
}

// Text returns the canonical serialization of the receiver's children (RFC 8).
// The receiver is treated as the root node and is not itself emitted.
//
// A name containing a control character other than TAB, LF or CR has no OGDL
// representation; Text omits such a node. Use Canonical to detect that case.
func (n *Node) Text() string {
	s, _ := n.Canonical()
	return s
}

// Canonical returns the canonical serialization of the receiver's children.
// It reports an error if any name cannot be represented (RFC 8.2).
func (n *Node) Canonical() (string, error) {
	var sb strings.Builder
	var err error
	for _, m := range n.Out {
		if e := m.write(0, &sb); e != nil && err == nil {
			err = e
		}
	}
	return sb.String(), err
}

func (n *Node) write(level int, sb *strings.Builder) error {
	name, err := formatName(n.Data)
	if err == nil {
		sb.WriteString(strings.Repeat("  ", level))
		sb.WriteString(name)
		sb.WriteRune('\n')
	}
	for _, m := range n.Out {
		if e := m.write(level+1, sb); e != nil && err == nil {
			err = e
		}
	}
	return err
}

// needsQuoting reports whether a name must be quoted in canonical form
// (RFC 8.2). Names not listed here MUST NOT be quoted.
func needsQuoting(name string) bool {
	if name == "" || name == `\` || name == "#" {
		return true
	}
	for i, r := range name {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return true
		}
		if i == 0 && (r == '"' || r == '\'') {
			return true
		}
	}
	return false
}

// formatName renders a node's data as an OGDL name in canonical form: its
// textual form, quoted if required. Only \\, \", \n and \t are escaped; the
// delimiter is always the double quote, so ' is never escaped.
func formatName(v interface{}) (string, error) {
	name := nameString(v)
	for _, r := range name {
		if isCharEnd(r) {
			return "", ErrControlCharacter
		}
	}
	if !needsQuoting(name) {
		return name, nil
	}
	return quoteName(name), nil
}

// quoteName writes name as a double-quoted string.
func quoteName(name string) string {
	var sb strings.Builder
	sb.WriteRune('"')
	for _, r := range name {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\t':
			sb.WriteString(`\t`)
		case '\r':
			sb.WriteString(`\n`)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteRune('"')
	return sb.String()
}
