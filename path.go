// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

// NewPath takes an Unicode string representing and OGDL path, into a Graph object.
//
// It also parses extended paths, as those used in templates, which may have
// argument lists.
func NewPath(s string) *Graph {
	parse := NewStringParser(s)
	parse.Path()
	return parse.GraphTop(TYPE_PATH)
}
