package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"unicode"

	"github.com/rveen/ogdl"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func main() {

	var optId bool

	flag.BoolVar(&optId, "id", false, "simplify @id's")

	flag.Parse()

	if flag.NArg() < 2 {
		println("usage\n  xml2ogdl <path> <file>")
		return
	}

	path := flag.Arg(0)

	b, err := ioutil.ReadFile(flag.Arg(1))
	if err != nil {
		return
	}

	g := xml2graph(b)

	if path != "." {
		g = g.Get(path)
	}

	if optId {
		simplify(g)
	}

	fmt.Printf("%s\n", g.Text())
}

func simplify(g *ogdl.Graph) {
	if g.Out == nil {
		return
	}

	for _, n := range g.Out {

		if len(n.Out) == 0 {
			continue
		}
		if n.Out[0].ThisString() == "@id" && len(n.Out) >= 2 {
			if n.This != "p" {
				n.This = n.Out[0].Out[0].This
			}
			n.Out[0] = n.Out[len(n.Out)-1]
			n.Out = n.Out[:1]
		}
		simplify(n)
	}
}

func xml2graph(b []byte) *ogdl.Graph {

	decoder := xml.NewDecoder(bytes.NewReader(b))

	g := ogdl.New(nil)
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
			if att {
				for _, at := range se.Attr {
					n.Add("@" + at.Name.Local).Add(at.Value)
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
