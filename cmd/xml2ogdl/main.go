package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"unicode"

	"github.com/rveen/ogdl"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func main() {
	if len(os.Args) < 2 {
		println("usage\n  xml2ogdl <file>")
		return
	}

	// If there is only one argument, than that is a path
	// If the path is just '.', return the whole graph
	// (in canonical form).
	//
	path := os.Args[1]

	b, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		return
	}

	g := xml2graph(b)

	if path != "." {
		g = g.Get(path)
	}

	fmt.Printf("%s\n", g.Text())
}

func xml2graph(b []byte) *ogdl.Graph {

	decoder := xml.NewDecoder(bytes.NewReader(b))

	g := ogdl.New()
	var key string
	level := -1

	att := true

	var stack []*ogdl.Graph
	stack = append(stack, g)

	tr := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

	for {
		// Read tokens from the XML document in a stream.
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		// Inspect the type of the token just read.
		switch se := t.(type) {

		case xml.StartElement:
			level++

			key = se.Name.Local
			// No accents in key
			key, _, _ = transform.String(tr, key)
			// - -> _
			key = strings.Replace(key, "-", "_", -1)

			n := stack[len(stack)-1].Add(key)
			// push
			stack = append(stack, n)
			if att && len(se.Attr) != 0 {
				a := n.Add("@")
				for _, at := range se.Attr {
					a.Add(at.Name.Local).Add(at.Value)
				}
			}

		case xml.CharData:

			val := strings.TrimSpace(string(se))
			if len(val) > 0 {
				stack[len(stack)-1].Add(val)
			}

		case xml.EndElement:
			level--
			// pop

			stack = stack[:len(stack)-1]

		}
	}

	return g
}
