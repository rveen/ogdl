package ogdl

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"sync"
)

// Log is an OGDL log (rfc/ogdl-binary-rfc.md, section 8): binary streams,
// appended one after another, each found again by its position, the offset
// of its first octet.
//
// The zero Log keeps its streams in memory (see Bytes); OpenLog keeps them in
// a file. A Log is safe for concurrent use.
type Log struct {
	mu       sync.Mutex
	f        *os.File
	noSync   bool
	mem      bytes.Buffer
	MaxDepth int // passed to the BinaryReader of Get; zero means DefaultMaxDepth
}

// OpenLog opens a log file for reading and appending, creating it if needed.
// By default every Add is synced to disk; see SetSync.
func OpenLog(file string) (*Log, error) {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0o666)
	if err != nil {
		return nil, err
	}
	return &Log{f: f}, nil
}

// Close closes the log file. Closing an in-memory log does nothing.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

// SetSync sets whether each Add and AddBinary syncs the file to disk (the
// default, true).
func (l *Log) SetSync(sync bool) {
	l.mu.Lock()
	l.noSync = !sync
	l.mu.Unlock()
}

// Sync commits the log file to disk.
func (l *Log) Sync() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	return l.f.Sync()
}

// Add appends the binary stream of n and returns its position.
func (l *Log) Add(n *Node) (int64, error) {
	b, err := n.MarshalBinary()
	if err != nil {
		return 0, err
	}
	return l.AddBinary(b)
}

// AddBinary appends b, which must be one binary stream, and returns its
// position. b is checked: a log only ever holds well-formed streams.
func (l *Log) AddBinary(b []byte) (int64, error) {
	if _, err := ParseBinary(b); err != nil {
		return 0, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.f == nil {
		pos := int64(l.mem.Len())
		l.mem.Write(b)
		return pos, nil
	}

	pos, err := l.f.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	if _, err := l.f.Write(b); err != nil {
		// Leave no partial stream behind for the next reader.
		l.f.Truncate(pos)
		return 0, err
	}
	if !l.noSync {
		if err := l.f.Sync(); err != nil {
			return 0, err
		}
	}
	return pos, nil
}

// Bytes returns the content of an in-memory log, and nil for a file log.
func (l *Log) Bytes() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		return nil
	}
	return l.mem.Bytes()
}

// Get decodes the stream at pos and returns it with the position of the next
// one. At the end of the log it returns io.EOF.
func (l *Log) Get(pos int64) (*Node, int64, error) {
	var n *Node
	next, err := l.read(pos, func(r *BinaryReader) error {
		var err error
		n, err = r.Read()
		return err
	})
	if err != nil {
		return nil, pos, err
	}
	return n, next, nil
}

// GetBinary returns the stream at pos as it is stored, after checking it,
// with the position of the next one. At the end of the log it returns io.EOF.
func (l *Log) GetBinary(pos int64) ([]byte, int64, error) {
	var raw bytes.Buffer
	next, err := l.read(pos, func(r *BinaryReader) error {
		r.r = teeByteReader{r.r, &raw}
		_, err := r.Read()
		return err
	})
	if err != nil {
		return nil, pos, err
	}
	return raw.Bytes(), next, nil
}

// read runs decode on a BinaryReader positioned at pos and returns the
// position after what it consumed.
func (l *Log) read(pos int64, decode func(*BinaryReader) error) (int64, error) {
	if pos < 0 {
		return pos, errors.New("ogdl: negative log position")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var src io.Reader
	if l.f == nil {
		if pos > int64(l.mem.Len()) {
			return pos, io.EOF
		}
		src = bytes.NewReader(l.mem.Bytes()[pos:])
	} else {
		src = io.NewSectionReader(l.f, pos, 1<<62)
	}

	r := &BinaryReader{r: bufio.NewReader(src), MaxDepth: l.MaxDepth}
	if err := decode(r); err != nil {
		var be *BinaryError
		if errors.As(err, &be) {
			be.Offset += pos // report the offset in the log
		}
		return pos, err
	}
	return pos + r.Offset(), nil
}

// teeByteReader copies every octet read to w.
type teeByteReader struct {
	r io.ByteReader
	w *bytes.Buffer
}

func (t teeByteReader) ReadByte() (byte, error) {
	c, err := t.r.ReadByte()
	if err == nil {
		t.w.WriteByte(c)
	}
	return c, err
}
