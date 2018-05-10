// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.package ogdl

package ogdl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// FromJSON converts a JSON text stream into OGDL.
//
// Json types returned by Decode/Unmarshall:
// - bool, for JSON booleans
// - int64 / float64, for JSON numbers
// - string, for JSON strings
// - []interface{}, for JSON arrays
// - map[string]interface{}, for JSON objects
// - nil for JSON null
//
// TODO I'm not sure about not giving lists a root node, but we need to avoid both
// useless nesting and also post-simplification (and its unwanted side effects).
// But, for example [ "a", [ "b", "c" ] ] will be returned as:
//     a
//     b
//     c
//
func FromJSON(buf []byte) (*Graph, error) {

	var v interface{}

	// Use Decoder, since we want to treat integers as integers.
	dec := json.NewDecoder(bytes.NewReader(buf))
	dec.UseNumber()

	// We expect here only one map or list
	err := dec.Decode(&v)
	if err != nil {
		return nil, err
	}

	return toGraph(v).GetAt(0), nil
}

func toGraph(v interface{}) *Graph {

	g := New("_")

	switch v.(type) {

	case []interface{}:
		// n := g.Add("_")
		for _, i := range v.([]interface{}) {
			g.Add(toGraph(i))
		}
	case map[string]interface{}:
		n := g.Add("_")
		for k, i := range v.(map[string]interface{}) {
			n.Add(k).AddNodes(toGraph(i))
		}
	case json.Number:
		// try firts to decode the number as an integer.
		i, err := v.(json.Number).Int64()
		if err != nil {
			f, err := v.(json.Number).Float64()
			if err != nil {
				f = math.NaN()
			}
			g.Add(f)
		} else {
			g.Add(i)
		}

	default:
		g.Add(v)
	}
	return g
}

// JSON produces JSON text from a Graph
//
// JSON has maps (objects, {}), lists (arrays, []) and values.
//
// Values can be strings, numbers, maps, lists, 'true', 'false' or 'null'
//
// map ::= '{' string ':' value [',' string : value]* '}'
// list ::= '[' value [',' value]* ']'
//
// By definition, since maps and lists cannot be distinguished in OGDL, any list
// should have a '_' root node.
// Any non-leaf node is a map (unless is contains '_', obvously).
//
func (g *Graph) JSON() []byte {

	var buf *bytes.Buffer
	buf = new(bytes.Buffer)

	g.json(buf)

	return buf.Bytes()
}

func (g *Graph) json(buf *bytes.Buffer) {

	// If this is a leaf node, print it as value
	if g.Len() == 0 {
		g.writeValue(buf)
		return
	}

	typ := 0

	// Now, it is either a map or a list
	if g.ThisString() == "_" {
		typ = '['
		buf.WriteString("[ ")
	} else if g.Len() > 1 {
		typ = '{'
		buf.WriteString("{ ")
	}

	comma := false

	for _, n := range g.Out {
		if comma {
			buf.WriteString(", ")
		}
		switch n.Len() {
		case 0:
			n.writeValue(buf)
		case 1:
			n.writeString(buf)
			buf.WriteString(": ")
			n.json(buf)
			comma = true
		default:
			n.json(buf)
			comma = true
		}
	}

	switch typ {
	case '[':
		buf.WriteByte(']')
	case '{':
		buf.WriteByte('}')
	}
}

func (g *Graph) writeString(buf *bytes.Buffer) {
	s := g.ThisString()
	buf.WriteByte('"')
	s = strings.Replace(s, "\"", "\\\"", -1)
	buf.WriteString(s)
	buf.WriteByte('"')
	buf.WriteByte(' ')
}

func (g *Graph) writeValue(buf *bytes.Buffer) {
	n, ok := _int64(g.This)
	if ok {
		buf.WriteString(fmt.Sprintf("%ld ", n))
		return
	}
	f, ok := _float64(g.This)
	if ok {
		buf.WriteString(fmt.Sprintf("%lf ", f))
		return
	}
	g.writeString(buf)
}
