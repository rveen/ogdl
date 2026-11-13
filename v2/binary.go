package ogdl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

// Binary OGDL: rfc/ogdl-binary-rfc.md.

var (
	ErrBinaryHeader  = errors.New("not an OGDL binary stream (bad header)")
	ErrBinaryInteger = errors.New("ill-formed integer")
	ErrBinaryLevel   = errors.New("level out of range")
	ErrBinaryText    = errors.New("text name cannot be encoded (contains 0x00, or begins with 0x01)")
)

// BinaryError reports an ill-formed binary stream, with the offset (from the
// start of the stream) of the octet at which it was found.
type BinaryError struct {
	Offset int64
	Err    error
}

func (e *BinaryError) Error() string {
	return fmt.Sprintf("ogdl: binary, offset %d: %v", e.Offset, e.Err)
}

func (e *BinaryError) Unwrap() error { return e.Err }

// maxBinaryInt is the largest integer the binary encoding can hold (RFC 4).
const maxBinaryInt = 0xFFFFFFF

// MarshalBinary encodes the children of n (n itself is the unnamed root) as
// an OGDL binary stream. A []byte name is written as data, any other name as
// text. It fails with ErrBinaryText for a text name containing 0x00 or
// beginning with 0x01.
func (n *Node) MarshalBinary() ([]byte, error) {
	buf := []byte{0x01, 'G', 0x00}
	if n == nil {
		// Like every accessor, a nil node is an empty tree.
		return append(buf, 0x00), nil
	}
	var err error
	for _, c := range n.Out {
		if buf, err = c.appendBinary(buf, 1); err != nil {
			return nil, err
		}
	}
	return append(buf, 0x00), nil
}

// Binary is MarshalBinary without the error: it returns nil when the tree
// cannot be encoded.
func (n *Node) Binary() []byte {
	b, err := n.MarshalBinary()
	if err != nil {
		return nil
	}
	return b
}

// WriteBinary writes the binary stream of n to w.
func (n *Node) WriteBinary(w io.Writer) error {
	b, err := n.MarshalBinary()
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (n *Node) appendBinary(buf []byte, level int) ([]byte, error) {
	if level > maxBinaryInt {
		return nil, ErrTooDeep
	}
	buf = appendBinaryInt(buf, level)

	if data, ok := n.Data.([]byte); ok {
		buf = append(buf, 0x01)
		for len(data) > 0 {
			k := len(data)
			if k > maxBinaryInt {
				k = maxBinaryInt
			}
			buf = appendBinaryInt(buf, k)
			buf = append(buf, data[:k]...)
			data = data[k:]
		}
		buf = append(buf, 0x00)
	} else {
		s := nameString(n.Data)
		if strings0(s) {
			return nil, ErrBinaryText
		}
		buf = append(buf, s...)
		buf = append(buf, 0x00)
	}

	var err error
	for _, c := range n.Out {
		if buf, err = c.appendBinary(buf, level+1); err != nil {
			return nil, err
		}
	}
	return buf, nil
}

// strings0 reports whether s cannot be a text record: it contains 0x00, or
// begins with 0x01.
func strings0(s string) bool {
	if len(s) > 0 && s[0] == 0x01 {
		return true
	}
	for i := 0; i < len(s); i++ {
		if s[i] == 0x00 {
			return true
		}
	}
	return false
}

// appendBinaryInt appends the shortest encoding of v (0 <= v <= maxBinaryInt).
func appendBinaryInt(buf []byte, v int) []byte {
	switch {
	case v < 0x80:
		return append(buf, byte(v))
	case v < 0x4000:
		return append(buf, byte(v>>8)|0x80, byte(v))
	case v < 0x200000:
		return append(buf, byte(v>>16)|0xc0, byte(v>>8), byte(v))
	default:
		return append(buf, byte(v>>24)|0xe0, byte(v>>16), byte(v>>8), byte(v))
	}
}

// ParseBinary decodes b, which must hold exactly one binary stream, and
// returns its unnamed root.
func ParseBinary(b []byte) (*Node, error) {
	r := NewBinaryReader(bytes.NewReader(b))
	n, err := r.Read()
	if err == io.EOF {
		return nil, &BinaryError{0, io.ErrUnexpectedEOF}
	}
	if err != nil {
		return nil, err
	}
	if r.Offset() != int64(len(b)) {
		return nil, &BinaryError{r.Offset(), errors.New("data after the end of the stream")}
	}
	return n, nil
}

// BinaryReader decodes a sequence of concatenated binary streams (RFC 8),
// one per call to Read.
type BinaryReader struct {
	// MaxDepth bounds the level of a record (RFC 6). Zero means
	// DefaultMaxDepth.
	MaxDepth int

	r   io.ByteReader
	off int64 // octets consumed in total
	// start is the offset at which the current stream began.
	start int64
}

// NewBinaryReader returns a reader of the streams in r. It reads no further
// than the end of each stream it returns only if r is an io.ByteReader (a
// *bufio.Reader or *bytes.Reader, for instance); any other reader is wrapped
// in a bufio.Reader, which may read ahead.
func NewBinaryReader(r io.Reader) *BinaryReader {
	br, ok := r.(io.ByteReader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &BinaryReader{r: br}
}

// Offset is the number of octets consumed so far: after a successful Read,
// the offset of the octet following the stream.
func (r *BinaryReader) Offset() int64 { return r.off }

// ReadBinary decodes one stream from r. See BinaryReader.Read.
func ReadBinary(r io.ByteReader) (*Node, error) {
	return (&BinaryReader{r: r}).Read()
}

// Read decodes the next stream and returns its unnamed root. At the end of
// the input, where a next stream would begin, it returns io.EOF. Any other
// problem is a *BinaryError, and no tree is returned.
func (r *BinaryReader) Read() (*Node, error) {
	r.start = r.off

	c, err := r.r.ReadByte()
	if err == io.EOF {
		return nil, io.EOF
	}
	if err != nil {
		return nil, err
	}
	r.off++
	if c != 0x01 {
		return nil, r.fail(ErrBinaryHeader)
	}
	for _, want := range []byte{'G', 0x00} {
		c, err := r.byte()
		if err != nil {
			return nil, err
		}
		if c != want {
			return nil, r.fail(ErrBinaryHeader)
		}
	}

	max := r.MaxDepth
	if max <= 0 {
		max = DefaultMaxDepth
	}

	root := &Node{}
	stack := []*Node{root} // stack[L] is the most recent node of level L
	for {
		at := r.off
		level, err := r.integer()
		if err != nil {
			return nil, err
		}
		if level == 0 {
			return root, nil
		}
		if level > len(stack) {
			return nil, &BinaryError{at - r.start, ErrBinaryLevel}
		}
		if level > max {
			return nil, &BinaryError{at - r.start, ErrTooDeep}
		}

		node, err := r.record()
		if err != nil {
			return nil, err
		}
		parent := stack[level-1]
		parent.Out = append(parent.Out, node)
		stack = append(stack[:level], node)
	}
}

// record reads a record's content, after its level.
func (r *BinaryReader) record() (*Node, error) {
	c, err := r.byte()
	if err != nil {
		return nil, err
	}

	if c == 0x01 { // data
		data := []byte{}
		for {
			k, err := r.integer()
			if err != nil {
				return nil, err
			}
			if k == 0 {
				return &Node{Data: data}, nil
			}
			// Grown as octets arrive, never from the announced length
			// (RFC 11).
			for ; k > 0; k-- {
				b, err := r.byte()
				if err != nil {
					return nil, err
				}
				data = append(data, b)
			}
		}
	}

	var text []byte
	for c != 0x00 {
		text = append(text, c)
		if c, err = r.byte(); err != nil {
			return nil, err
		}
	}
	return &Node{Data: string(text)}, nil
}

// integer reads an integer (RFC 4).
func (r *BinaryReader) integer() (int, error) {
	at := r.off
	b0, err := r.byte()
	if err != nil {
		return 0, err
	}
	var extra int
	var v int
	switch {
	case b0 < 0x80:
		return int(b0), nil
	case b0 < 0xc0:
		extra, v = 1, int(b0&0x3f)
	case b0 < 0xe0:
		extra, v = 2, int(b0&0x1f)
	case b0 < 0xf0:
		extra, v = 3, int(b0&0x0f)
	default:
		return 0, &BinaryError{at - r.start, ErrBinaryInteger}
	}
	for ; extra > 0; extra-- {
		b, err := r.byte()
		if err != nil {
			return 0, err
		}
		v = v<<8 | int(b)
	}
	return v, nil
}

// byte reads one octet inside a stream: the end of input there is an error.
func (r *BinaryReader) byte() (byte, error) {
	c, err := r.r.ReadByte()
	if err == io.EOF {
		return 0, r.fail(io.ErrUnexpectedEOF)
	}
	if err != nil {
		return 0, err
	}
	r.off++
	return c, nil
}

func (r *BinaryReader) fail(err error) error {
	return &BinaryError{r.off - r.start, err}
}
