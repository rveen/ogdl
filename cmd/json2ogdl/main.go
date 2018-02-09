// Copyright 2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// json2ogdl [json_file]*
//
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rveen/ogdl"
)

func main() {

	var err error

	flag.Parse()

	source := os.Stdin

	if len(flag.Args()) > 0 {
		source, err = os.Open(flag.Args()[0])
		defer source.Close()
		if err != nil {
			println(err.Error())
			return
		}
	}

	var json []byte
	buf := make([]byte, 4096)

	for {
		n, err := source.Read(buf)
		if err != nil || n == 0 {
			break
		}
		json = append(json, buf[:n]...)
	}

	g, _ := ogdl.FromJSON(json)

	fmt.Println(g.Text())
}
