package ogdl

import (
	"bytes"
	"testing"
	"time"
)

func TestBinParser1(t *testing.T) {

	// newVarInt
	b := newVarInt(0x3fff)
	p := newBytesBinParser(b)
	i := p.varInt()
	if i != 0x3fff || len(b) != 2 {
		t.Error("varInt 0x3fff")
	}

	b = newVarInt(0x4000)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 0x4000 || len(b) != 3 {
		t.Error("varInt 0x4000")
	}

	b = newVarInt(0x1fffff)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 0x1fffff || len(b) != 3 {
		t.Error("varInt 0x1fffff", i)
	}

	b = newVarInt(0xfffffff)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 0xfffffff || len(b) != 4 {
		t.Error("varInt 0xfffffff", i)
	}

	b = newVarInt(127)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 127 || len(b) != 1 {
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

// A binary node that is not the last node in the stream must not swallow the
// nodes that follow it. Per the BNF a zero length ends a binary node:
//
//	binary-node ::= 0x01 ( length data )* 0x00
func TestBinNodeTerminates(t *testing.T) {

	r := []byte{1, 'G', 0,
		1, 1, 1, 0x55, 0, // level 1, binary node "U", zero length ends it
		1, 'b', 0, // level 1, text node "b"
		0}

	g := FromBinary(r)

	if g.Len() != 2 {
		t.Fatalf("want 2 root nodes, got %d: %s", g.Len(), g.Text())
	}
	if g.Out[0].ThisString() != "U" || g.Out[1].ThisString() != "b" {
		t.Errorf("got %q, %q", g.Out[0].ThisString(), g.Out[1].ThisString())
	}
}

// newVarInt and varInt must agree over the whole 28-bit range.
func TestVarIntRoundTrip(t *testing.T) {

	for _, i := range []int{0, 1, 0x7f, 0x80, 0x3fff, 0x4000,
		0x1fffff, 0x200000, 0xabcdef, 0x1234567, 0xfffffff} {

		b := newVarInt(i)
		if b == nil {
			t.Errorf("newVarInt(%#x) = nil", i)
			continue
		}
		p := newBytesBinParser(b)
		if got := p.varInt(); got != i {
			t.Errorf("varInt round trip: wrote %#x, read %#x (%#v)", i, got, b)
		}
	}
}

// A []byte node must survive a round trip as a []byte node, interior NUL
// bytes included.
func TestBinaryNodeRoundTrip(t *testing.T) {

	payload := []byte{0x00, 0x01, 0xff, 0x00, 'x'}

	g := New(nil)
	g.Add("can0").Add(payload)

	n := FromBinary(g.Binary()).Get("can0")

	if n.Len() != 1 {
		t.Fatalf("want 1 child under can0, got %d", n.Len())
	}
	got, ok := n.Out[0].This.([]byte)
	if !ok {
		t.Fatalf("child is %T, want []byte", n.Out[0].This)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("payload: got %#v, want %#v", got, payload)
	}
}

// A truncated stream must terminate, not spin on EOF.
func TestTruncatedStream(t *testing.T) {

	for _, r := range [][]byte{
		{1, 'G', 0, 1, 'a', 'b', 'c'},    // text node, no terminating NUL
		{1, 'G', 0, 1, 1, 4, 0xaa, 0xbb}, // binary node, length past end
	} {
		done := make(chan bool, 1)
		go func() { FromBinary(r); done <- true }()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("FromBinary did not terminate on %#v", r)
		}
	}
}
