package ogdl

import (
	"fmt"
	"strings"
	"testing"
)

const input = `network
  eth0
    ip   192.168.0.10
    mask 255.255.255.0
    gw   192.168.0.1
    title 'hola hola'

hostname crispin
description \
  some long text
  and some more.`

func TestLexer(t *testing.T) {
	lx := NewLexer(strings.NewReader(input))

	for {
		t, err := lx.NextToken()
		if t.Type == TokenEOF || err != nil {
			break
		}
		fmt.Printf("%v [%s]\n", t.StringType(), t.Value)
	}
}
