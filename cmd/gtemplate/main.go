// Copyright 2017, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// gtemplate <template_file> [ogdl_file]*
//
// Processes a template file and solves variables in it using data from any of
// the OGDL files given
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rveen/ogdl"
)

func main() {

	flag.Usage = p
	flag.Parse()

	if len(flag.Args()) < 1 {
		println("missing template file")
		return
	}

	var data *ogdl.Graph

	t, _ := ioutil.ReadFile(flag.Args()[0])
	tpl := ogdl.NewTemplate(string(t))

	for i, f := range flag.Args() {
		switch i {
		case 0:
		case 1:
			data = ogdl.FromFile(f)
		default:
			g := ogdl.FromFile(f)
			data.AddNodes(g)
		}
	}

	b := tpl.Process(data)
	fmt.Println(string(b))
}

func p() {
	fmt.Fprintf(os.Stderr, "Usage: gtemplate <template>  [ogdl_file]*\n")
	flag.PrintDefaults()
}
