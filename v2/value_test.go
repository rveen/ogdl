package ogdl

import (
	"reflect"
	"testing"
)

// TestAccessors covers Int64, Float64, Bool and Strings: each reads the text
// below a node, and falls back to the default when there is none to read.
func TestAccessors(t *testing.T) {
	g, err := ParseString(`port 8080
ratio 0.75
neg -3
word abc
on true
off false
tags
  web
  api
`)
	if err != nil {
		t.Fatal(err)
	}

	ints := []struct {
		path string
		def  []int64
		want int64
	}{
		{"port", nil, 8080},
		{"neg", nil, -3},
		{"word", nil, 0},
		{"word", []int64{9}, 9},
		{"missing", []int64{9}, 9},
		{"ratio", []int64{9}, 0}, // a float truncates; it is not missing
	}
	for _, c := range ints {
		if got := g.Get(c.path).Int64(c.def...); got != c.want {
			t.Errorf("%s.Int64(%v) = %d, want %d", c.path, c.def, got, c.want)
		}
	}

	floats := []struct {
		path string
		def  []float64
		want float64
	}{
		{"ratio", nil, 0.75},
		{"port", nil, 8080},
		{"word", nil, 0},
		{"word", []float64{1.5}, 1.5},
		{"missing", []float64{1.5}, 1.5},
	}
	for _, c := range floats {
		if got := g.Get(c.path).Float64(c.def...); got != c.want {
			t.Errorf("%s.Float64(%v) = %g, want %g", c.path, c.def, got, c.want)
		}
	}

	bools := []struct {
		path string
		def  []bool
		want bool
	}{
		{"on", nil, true},
		{"off", nil, false},
		{"off", []bool{true}, false},
		{"missing", nil, false},
		{"missing", []bool{true}, true},
	}
	for _, c := range bools {
		if got := g.Get(c.path).Bool(c.def...); got != c.want {
			t.Errorf("%s.Bool(%v) = %v, want %v", c.path, c.def, got, c.want)
		}
	}

	if got := g.Get("tags").Strings(); !reflect.DeepEqual(got, []string{"web", "api"}) {
		t.Errorf("tags.Strings() = %q", got)
	}
	if got := g.Get("missing").Strings(); len(got) != 0 {
		t.Errorf("missing.Strings() = %q, want empty", got)
	}
	var nilNode *Node
	if nilNode.Strings() != nil || nilNode.Int64(5) != 5 || nilNode.Float64(5) != 5 || !nilNode.Bool(true) {
		t.Error("accessors on a nil node must return the default")
	}

	// Non-string data, as built in code, reads the same way.
	n := &Node{Out: []*Node{{Data: 42}}}
	if n.Int64() != 42 || n.Float64() != 42 {
		t.Errorf("Int64/Float64 of int data = %d, %g", n.Int64(), n.Float64())
	}
}
