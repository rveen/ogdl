package ogdl

import (
	"bytes"
	"testing"
)

func TestBinParser1(t *testing.T) {

	// newVarInt
	b := newVarInt(0x3fff)
	p := newBytesBinParser(b)
	i := p.varInt()
	if i != 0x3fff && len(b) != 2 {
		t.Error("varInt 0x3fff")
	}

	b = newVarInt(0x4000)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 0x4000 && len(b) != 3 {
		t.Error("varInt 0x4000")
	}

	b = newVarInt(0x1fffff)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 0x1fffff && len(b) != 4 {
		t.Error("varInt 0x1fffff", i)
	}

	b = newVarInt(0xfffffff)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 0xfffffff && len(b) != 4 {
		t.Error("varInt 0xfffffff", i)
	}

	b = newVarInt(127)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 127 && len(b) != 1 {
		t.Error("varInt 127")
	}

	b = newVarInt(-1)
	if b != nil {
		t.Error("newVarInt -1")
	}
	b = newVarInt(0x10000000)
	if b != nil {
		t.Error("newVarInt to high")
	}

	// force incorrect header
	h := []byte{1, 'G', 0}
	h2 := []byte{0, 'G', 0}
	h3 := []byte{1, 'H', 0}
	h4 := []byte{1, 'G', 1}

	p = newBytesBinParser(h)
	if !p.header() {
		t.Error("header")
	}
	p = newBytesBinParser(h2)
	if p.header() {
		t.Error("header")
	}
	p = newBytesBinParser(h3)
	if p.header() {
		t.Error("header")
	}
	p = newBytesBinParser(h4)
	if p.header() {
		t.Error("header")
	}
}

func TestBinParser2(t *testing.T) {

	s := "a"

	p := newBinParser(bytes.NewReader([]byte(s)))

	c := p.read()
	if c != 'a' {
		t.Error("read error")
	}

	p.unread()

	c = p.read()
	if c != 'a' {
		t.Error("unread error")
	}

	c = p.read()
	if c != -1 {
		t.Error("EOS read error")
	}
}

func TestBinParser3(t *testing.T) {

	r := []byte{1, 'G', 0, 1, 'a', 0, 2, 'b', 0, 0}

	// Starting from a NilGraph
	g := New(nil)
	b := g.Binary()

	if len(b) != 4 {
		t.Error("Binary() on NilGraph")
	}

	// Starting from a NilGraph
	g = New(nil)
	g.Add("a").Add("b")
	b = g.Binary()

	if string(r) != string(b) {

		for i := 0; i < len(b); i++ {
			println(b[i])
		}

		t.Error("Binary() failed")
	}

	var nul *Graph
	b = nul.Binary()
	if b != nil {
		t.Error("Binary nil failed")
	}

	g = FromBinary(r)
	// g = g.Out[0]

	if g.Len() != 1 {
		t.Error("BinParse() failed")
	}
	if g.String() != "a" {
		t.Error("BinParse() failed")
	}
}

func TestBinParser4(t *testing.T) {

	r := []byte{1, 'G', 0 /*lev*/, 1 /*bin*/, 1 /* len */, 1, 0x55 /*end bin */, 0, 0}

	g := FromBinary(r)
	//g = g.Out[0]

	if g.Len() != 1 {
		t.Error("BinParse() failed")
	}
	if g.String() != "U" {
		t.Error("BinParse() failed")
	}
}
