// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
)

// BinParser and its methods implement a parser for binary OGDL, as defined in the
// specification available at ogdl.org, reproduced below.
// 
// ogdl-binary ::= header ( level node )* 0x00
//
// header ::= 0x01 'G' 0x00
// level  ::= varInt
// node   ::= text-node | binary-node
//
// text-node ::= text 0x00
// binary-node ::= 0x01 ( length data )* 0x00
//
// length ::= multibyte-integer
// data :: byte[length]
type BinParser struct {
	r    *bufio.Reader
	ix   int
	last int
	// Used to count bytes read.
	N int
}

//NewBytesBinParser creates a parser that can convert a binary OGDL byte stream into an 
// ogdl.Graph object. To actually parse the stream, the method Parse() has to be invoked.
func NewBytesBinParser(b []byte) *BinParser {
	return &BinParser{bufio.NewReader(bytes.NewReader(b)), 0, 0, 0}
}

//NewBytesBinParser creates a parser that can convert a binary OGDL file into an 
// ogdl.Graph object. To actually parse the stream, the method Parse() has to be invoked.
func NewFileBinParser(file string) *BinParser {

	// Read the entire file into memory
	b, err := ioutil.ReadFile(file)
	if err != nil || len(b) == 0 {
		return nil
	}

	return NewBytesBinParser(b)
}

//NewBytesBinParser creates a parser that can convert a binary OGDL source into an 
// ogdl.Graph object. To actually parse the stream, the method Parse() has to be invoked.
func NewBinParser(r io.Reader) *BinParser {
	return &BinParser{bufio.NewReader(r), 0, 0, 0}
}

// BinParse converts an OGDL binary stream of bytes into a Graph.
func BinParse(b []byte) *Graph {
	p := NewBytesBinParser(b)
	return p.Parse()
}

// Binary converts a Graph to a binary OGDL byte stream.
func (g *Graph) Binary() []byte {

	if g == nil {
		return nil
	}

	// Header
	buf := make([]byte, 3)
	buf[0] = 1
	buf[1] = 'G'
	buf[2] = 0

	buf = g.bin(1, buf)

	// Ending null
	buf = append(buf, 0)

	return buf
}

func (g *Graph) bin(level int, buf []byte) []byte {

	// Skip empty nodes
	if len(g.String()) != 0 {
		buf = append(buf, newVarInt(level)...)
		buf = append(buf, g.Bytes()...)
		buf = append(buf, 0)
		level++
	}

	for _, node := range g.Out {
		buf = node.bin(level, buf)
	}

	return buf
}

// Graph parses a binary OGDL stream and returns a Graph.
func (p *BinParser) Parse() *Graph {

	if !p.header() {
		return nil
	}

	ev := NewEventHandler()

	for {
		// BUG(): blobs not handled
		lev, _, b := p.line(true)
		if lev == 0 {
			break
		}
		ev.AddAt(string(b), lev)
	}
	return ev.Graph()
}

// newVarInt produces a variable integer from an int.
func newVarInt(i int) []byte {

	if i < 0x80 {
		b := make([]byte, 1)
		b[0] = byte(i)
		return b
	}
	
	if i < 0x4000 {
	    b := make([]byte, 2)
		b[0] = byte(i>>8 | 0x80)
		b[1] = byte(i&0xff)
		return b
	} 
	
	if i < 0x200000 {
	    b := make([]byte, 3)
		b[0] = byte(i>>16 | 0xc0)
		b[1] = byte(i>>8 & 0xff)
		b[2] = byte(i & 0xff)
		return b
	}
	
	return nil
}

/* ---------------------------------------------
   Productions:

   - header
   - varint
   - line

   A binary OGDL file is a sequence of Lines:

     Binary_OGDL := Line+

   where the first Line is a Header.
   --------------------------------------------- */

// header is the parser production that reads the header from the stream
//
// header ::= 0x01 'G' 0x00
func (p *BinParser) header() bool {

	if p.read() != 1 {
		return false
	}
	if p.read() != 'G' {
		return false
	}
	if p.read() != 0 {
		return false
	}
	return true
}



// varInt is the parser production that reads a variable length integer from the stream.
//
// varInt ::=  
//   0x00 - 0x7F:      0xxxxxxx
//   0x00 - 0x3FFF:    10xxxxxx xxxxxxxx
//   0x00 - 0x1FFFFF:  110xxxxx xxxxxxxx xxxxxxxx
//   0x00 - 0xFFFFFFF: 1110xxxx xxxxxxxx xxxxxxxx xxxxxxxx
//    ...      
func (p *BinParser) varInt() int {

	b0 := p.read()

	if b0 < 0x80 {
		return b0
	}

	if b0 < 0xc0 {
		b1 := p.read()
		return (b0 & 0x3f)<<8 | b1
	}

	if b0 < 0xe0 {
		b1 := p.read()
		b2 := p.read()
		return ((b0 & 0x1f) << 16) | (b1 << 8) | b2
	}

	if b0 < 0xf0 {
		b1 := p.read()
		b2 := p.read()
		b3 := p.read()
		return ((b0 & 0x0f) << 24) | (b1 << 16) | (b2 << 8) | b3
	}

	return -1
}

// line is the parser production that reads one line out of the binary OGDL stream.
//
// line ::= level node 0x00 
// where
//   level ::= varInt
//   node  ::= text-node | binary-node
//
func (p *BinParser) line(write bool) ( /* level */ int /* blob*/, bool, []byte) {

	// Read int
	level := p.varInt()
	if level < 1 {
		return 0, false, nil
	}

	// Binary node?
	n := p.read()
	if n == 1 {
		// Read bytes...
		return level, true, nil
	}

	// Text node. Read bytes until 0
	buf := bytes.Buffer{}
	buf.WriteByte(byte(n))

	for {
		c := p.read()
		if c == 0 {
			return level, false, buf.Bytes()
		}
		if write {
			buf.WriteByte(byte(c))
		}
	}
}

/* ---------------------------------------------
   Elementary byte handling
   --------------------------------------------- */

// Read reads one character (byte) from the stream, returning it in the for of an int.
// Returning an int permits signaling and EOS with -1.
func (p *BinParser) read() int {

	i, err := p.r.ReadByte()

	c := int(i)
	if err == io.EOF {
		c = -1
	} else {
		p.N++
	}

	p.last = c

	return c
}

// Unread puts the last character readed back into the stream.
func (p *BinParser) unread() {
	if p.last > 0 {
		p.N--
		p.r.UnreadByte()
	}
}
