package ogdl

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"math/rand"
	"strings"
	"testing"
)

// hx turns "01 47 00" into bytes.
func hx(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// tn builds a node: tn("a", tn("b")).
func tn(data any, out ...*Node) *Node { return &Node{Data: data, Out: out} }

// sameBinTree compares names, including their type (string or []byte), and
// structure.
func sameBinTree(a, b *Node) bool {
	switch x := a.Data.(type) {
	case []byte:
		y, ok := b.Data.([]byte)
		if !ok || !bytes.Equal(x, y) {
			return false
		}
	default:
		if _, isBytes := b.Data.([]byte); isBytes || nameString(a.Data) != nameString(b.Data) {
			return false
		}
	}
	if len(a.Out) != len(b.Out) {
		return false
	}
	for i := range a.Out {
		if !sameBinTree(a.Out[i], b.Out[i]) {
			return false
		}
	}
	return true
}

func deep(depth int, leaf string) *Node {
	root := &Node{}
	n := root
	for i := 1; i < depth; i++ {
		c := tn("x")
		n.Out = append(n.Out, c)
		n = c
	}
	n.Out = append(n.Out, tn(leaf))
	return root
}

// Appendix A. Encoding gives exactly these octets; decoding gives exactly
// this tree, with the same name types. The first five, the depth-128 and the
// 200-octet vectors are what github.com/rveen/ogdl v1.5 writes.
func TestBinaryVectors(t *testing.T) {
	data200 := bytes.Repeat([]byte{0x7a}, 200)
	depth128 := deep(128, "y")

	for _, v := range []struct {
		name   string
		tree   *Node
		stream []byte
	}{
		{"empty", tn(nil), hx(t, "01 47 00 00")},
		{"one node", tn(nil, tn("a")), hx(t, "01 47 00 01 61 00 00")},
		{"children", tn(nil, tn("a", tn("b"), tn("c")), tn("d")), hx(t, "01 47 00 01 61 00 02 62 00 02 63 00 01 64 00 00")},
		{"data child", tn(nil, tn("x", tn([]byte{0, 1, 2}))), hx(t, "01 47 00 01 78 00 02 01 03 00 01 02 00 00")},
		{"text 42", tn(nil, tn("n", tn("42"))), hx(t, "01 47 00 01 6e 00 02 34 32 00 00")},
		{"empty name", tn(nil, tn("", tn("child"))), hx(t, "01 47 00 01 00 02 63 68 69 6c 64 00 00")},
		{"empty data", tn(nil, tn([]byte{})), hx(t, "01 47 00 01 01 00 00")},
		{"200 octets", tn(nil, tn(data200)), append(append(hx(t, "01 47 00 01 01 80 c8"), data200...), 0, 0)},
	} {
		got, err := v.tree.MarshalBinary()
		if err != nil || !bytes.Equal(got, v.stream) {
			t.Errorf("%s: encode\n got % x, %v\nwant % x", v.name, got, err, v.stream)
		}
		n, err := ParseBinary(v.stream)
		if err != nil || !sameBinTree(n, v.tree) {
			t.Errorf("%s: decode: got %v, %v", v.name, n, err)
		}
	}

	b, err := depth128.MarshalBinary()
	if err != nil || !bytes.HasSuffix(b, hx(t, "7f 78 00 80 80 79 00 00")) || len(b) != 389 {
		t.Errorf("depth 128: len %d, tail % x, %v", len(b), b[len(b)-8:], err)
	}
	if n, err := ParseBinary(b); err != nil || !sameBinTree(n, depth128) {
		t.Errorf("depth 128: decode: %v", err)
	}

	// An integer is a typed value; it comes back as its text.
	if b, _ := tn(nil, tn("n", tn(42))).MarshalBinary(); !bytes.Equal(b, hx(t, "01 47 00 01 6e 00 02 34 32 00 00")) {
		t.Errorf("int 42: % x", b)
	}
}

// Appendix A.1.
func TestBinaryIllFormed(t *testing.T) {
	for _, v := range []struct {
		stream string
		want   error
	}{
		{"01 47", io.ErrUnexpectedEOF},
		{"01 48 00 00", ErrBinaryHeader},
		{"02 47 00 00", ErrBinaryHeader},
		{"01 47 00 01 61 00", io.ErrUnexpectedEOF},
		{"01 47 00 01 61", io.ErrUnexpectedEOF},
		{"01 47 00 02 61 00 00", ErrBinaryLevel},
		{"01 47 00 01 61 00 03 62 00 00", ErrBinaryLevel},
		{"01 47 00 f0 61 00 00", ErrBinaryInteger},
		{"01 47 00 01 01 05 61 62 00", io.ErrUnexpectedEOF},
		{"01 47 00 01 01 f5 00 00", ErrBinaryInteger},
	} {
		n, err := ParseBinary(hx(t, v.stream))
		var be *BinaryError
		if n != nil || !errors.Is(err, v.want) || !errors.As(err, &be) {
			t.Errorf("%s: got %v, %v; want %v as a *BinaryError", v.stream, n, err, v.want)
		}
	}

	// The empty input is the normal end of a sequence for a reader ...
	if n, err := NewBinaryReader(bytes.NewReader(nil)).Read(); n != nil || err != io.EOF {
		t.Errorf("reader, empty input: %v, %v; want io.EOF", n, err)
	}
	// ... but not a stream.
	if _, err := ParseBinary(nil); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("ParseBinary(nil): %v", err)
	}
	// One stream means exactly one.
	if _, err := ParseBinary(hx(t, "01 47 00 00 01")); err == nil {
		t.Error("ParseBinary accepted data after the stream")
	}
}

func TestBinaryOffsets(t *testing.T) {
	_, err := ParseBinary(hx(t, "01 47 00 01 61 00 03 62 00 00"))
	var be *BinaryError
	if !errors.As(err, &be) || be.Offset != 6 {
		t.Errorf("level jump: %v, want offset 6", err)
	}
}

func TestBinaryUnencodableText(t *testing.T) {
	for _, s := range []string{"a\x00b", "\x00", "\x01a"} {
		if b, err := tn(nil, tn(s)).MarshalBinary(); b != nil || !errors.Is(err, ErrBinaryText) {
			t.Errorf("%q: got % x, %v; want ErrBinaryText", s, b, err)
		}
	}
	// Elsewhere 0x01, and other control characters, are fine.
	tree := tn(nil, tn("a\x01b\x02"))
	b, err := tree.MarshalBinary()
	if n, err2 := ParseBinary(b); err != nil || err2 != nil || !sameBinTree(n, tree) {
		t.Errorf("control characters: %v, %v", err, err2)
	}
	if tn(nil, tn("\x01")).Binary() != nil {
		t.Error("Binary should be nil when MarshalBinary fails")
	}
}

func TestBinaryMaxDepth(t *testing.T) {
	b, _ := deep(10, "y").MarshalBinary()
	r := NewBinaryReader(bytes.NewReader(b))
	r.MaxDepth = 9
	if _, err := r.Read(); !errors.Is(err, ErrTooDeep) {
		t.Errorf("got %v, want ErrTooDeep", err)
	}
	b, _ = deep(DefaultMaxDepth+1, "y").MarshalBinary()
	if _, err := ParseBinary(b); !errors.Is(err, ErrTooDeep) {
		t.Errorf("default limit: got %v, want ErrTooDeep", err)
	}
}

// Concatenated streams (RFC 8): each Read stops at its stream's end, which
// is where the next begins.
func TestBinarySequence(t *testing.T) {
	trees := []*Node{tn(nil, tn("a")), tn(nil), tn(nil, tn("b", tn([]byte("c"))))}
	var all []byte
	var ends []int64
	for _, tr := range trees {
		b, _ := tr.MarshalBinary()
		all = append(all, b...)
		ends = append(ends, int64(len(all)))
	}

	br := bufio.NewReader(bytes.NewReader(all))
	r := NewBinaryReader(br)
	for i, want := range trees {
		n, err := r.Read()
		if err != nil || !sameBinTree(n, want) || r.Offset() != ends[i] {
			t.Fatalf("stream %d: %v, %v, offset %d want %d", i, n, err, r.Offset(), ends[i])
		}
	}
	if n, err := r.Read(); n != nil || err != io.EOF {
		t.Errorf("after the last stream: %v, %v; want io.EOF", n, err)
	}

	// ReadBinary on a ByteReader leaves the rest for the next call.
	br = bufio.NewReader(bytes.NewReader(all))
	for i, want := range trees {
		if n, err := ReadBinary(br); err != nil || !sameBinTree(n, want) {
			t.Fatalf("ReadBinary %d: %v", i, err)
		}
	}
}

// Random trees survive the round trip, with their name types.
func TestBinaryRoundTrip(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	var grow func(depth int) *Node
	grow = func(depth int) *Node {
		n := &Node{}
		switch rnd.Intn(4) {
		case 0:
			b := make([]byte, rnd.Intn(300))
			rnd.Read(b)
			n.Data = b
		case 1:
			n.Data = ""
		default:
			n.Data = strings.Repeat("ñ", rnd.Intn(5)) + string(rune('a'+rnd.Intn(26)))
		}
		if depth < 6 {
			for i := rnd.Intn(4); i > 0; i-- {
				n.Out = append(n.Out, grow(depth+1))
			}
		}
		return n
	}
	for i := 0; i < 200; i++ {
		tree := grow(0)
		tree.Data = nil
		b, err := tree.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		n, err := ParseBinary(b)
		if err != nil || !sameBinTree(n, tree) {
			t.Fatalf("tree %d: %v", i, err)
		}
	}
}

// Hostile lengths do not allocate in advance (RFC 11): a stream announcing a
// 256 MiB chunk with nothing behind it fails quickly.
func TestBinaryHostileLength(t *testing.T) {
	_, err := ParseBinary(hx(t, "01 47 00 01 01 ef ff ff ff"))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("got %v", err)
	}
}

func TestWriteBinary(t *testing.T) {
	var buf bytes.Buffer
	if err := tn(nil, tn("a")).WriteBinary(&buf); err != nil || !bytes.Equal(buf.Bytes(), hx(t, "01 47 00 01 61 00 00")) {
		t.Errorf("% x, %v", buf.Bytes(), err)
	}
	if err := tn(nil, tn("\x00")).WriteBinary(&buf); !errors.Is(err, ErrBinaryText) {
		t.Errorf("got %v", err)
	}
}
