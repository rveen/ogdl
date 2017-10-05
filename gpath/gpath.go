// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"ogdl"
	"os"
)

func main() {

	if len(os.Args) < 2 {
		println("usage\n  gpath <path> [file]")
		return
	}

	// If there is only one argument, than that is a path
	// If the path is just '.', return the whole graph
	// (in canonical form).
	//
	path := os.Args[1]

	// A second argument is a file name
	source := os.Stdin
	var err error = nil
	if len(os.Args) > 2 {
		source, err = os.Open(os.Args[2])
		if err != nil {
			println(err.Error())
			return
		}
	}

	g := ogdl.FromReader(source)

	r := g

	if path != "." {
		r = g.Get(path)
	}

	// add a final newline if not already there
	s := r.Text()
	if len(s) == 0 {
		return
	}
	c := s[len(s)-1]
	if c != 10 && c != 13 {
		s += "\n"
	}

	fmt.Print(s)
}
