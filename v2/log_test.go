package ogdl

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func logTrees() []*Node {
	return []*Node{
		tn(nil, tn("a", tn("b"))),
		tn(nil),
		tn(nil, tn("c", tn([]byte{0, 1}))),
	}
}

// walk reads a log from the start, returning the trees and their positions.
func walk(t *testing.T, l *Log) ([]*Node, []int64) {
	t.Helper()
	var trees []*Node
	var at []int64
	var pos int64
	for {
		n, next, err := l.Get(pos)
		if err == io.EOF {
			if next != pos {
				t.Errorf("at the end, next is %d, want %d", next, pos)
			}
			return trees, at
		}
		if err != nil {
			t.Fatalf("Get(%d): %v", pos, err)
		}
		trees = append(trees, n)
		at = append(at, pos)
		pos = next
	}
}

func checkLog(t *testing.T, l *Log) {
	t.Helper()
	var positions []int64
	for _, tr := range logTrees() {
		pos, err := l.Add(tr)
		if err != nil {
			t.Fatal(err)
		}
		positions = append(positions, pos)
	}
	trees, at := walk(t, l)
	if len(trees) != 3 {
		t.Fatalf("read %d trees, want 3", len(trees))
	}
	for i, want := range logTrees() {
		if !sameBinTree(trees[i], want) || at[i] != positions[i] {
			t.Errorf("tree %d at %d (added at %d) differs", i, at[i], positions[i])
		}
		// Each position is stable and can be read directly.
		raw, _, err := l.GetBinary(positions[i])
		wantRaw, _ := want.MarshalBinary()
		if err != nil || !bytes.Equal(raw, wantRaw) {
			t.Errorf("GetBinary(%d): % x, %v; want % x", positions[i], raw, err, wantRaw)
		}
	}
}

func TestLogMemory(t *testing.T) {
	var l Log
	checkLog(t, &l)
	// Bytes is the concatenation, readable as a sequence (gstore's .gb files).
	r := NewBinaryReader(bytes.NewReader(l.Bytes()))
	for i := 0; i < 3; i++ {
		if _, err := r.Read(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.Read(); err != io.EOF {
		t.Errorf("after three streams: %v", err)
	}
}

func TestLogFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "log.gb")
	l, err := OpenLog(file)
	if err != nil {
		t.Fatal(err)
	}
	checkLog(t, l)
	if l.Bytes() != nil {
		t.Error("Bytes of a file log should be nil")
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopened, the log has the same content, and appends after it.
	l, err = OpenLog(file)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	l.SetSync(false)
	if _, err := l.Add(tn(nil, tn("d"))); err != nil {
		t.Fatal(err)
	}
	if trees, _ := walk(t, l); len(trees) != 4 || trees[3].N(0).ThisString() != "d" {
		t.Errorf("after reopening: %d trees", len(trees))
	}
}

func TestLogRejects(t *testing.T) {
	var l Log
	if _, err := l.AddBinary([]byte{1, 'G', 0, 1, 'a', 0}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("AddBinary of a truncated stream: %v", err)
	}
	if _, err := l.Add(tn(nil, tn("\x00"))); !errors.Is(err, ErrBinaryText) {
		t.Errorf("Add of an unencodable tree: %v", err)
	}
	if len(l.Bytes()) != 0 {
		t.Error("a rejected stream was stored")
	}
	if _, _, err := l.Get(-1); err == nil {
		t.Error("Get(-1) succeeded")
	}
}

// A damaged file log reports the offset of the problem in the file.
func TestLogDamaged(t *testing.T) {
	file := filepath.Join(t.TempDir(), "log.gb")
	good, _ := tn(nil, tn("a")).MarshalBinary()
	if err := os.WriteFile(file, append(good, 1, 'G', 0, 1, 'b'), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := OpenLog(file)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	_, next, err := l.Get(0)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = l.Get(next)
	var be *BinaryError
	if !errors.As(err, &be) || !errors.Is(err, io.ErrUnexpectedEOF) || be.Offset != int64(len(good))+5 {
		t.Errorf("got %v, want an unexpected end at offset %d", err, len(good)+5)
	}
}

func TestLogConcurrent(t *testing.T) {
	l, err := OpenLog(filepath.Join(t.TempDir(), "log.gb"))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	l.SetSync(false)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if _, err := l.Add(tn(nil, tn("x", tn("y")))); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if trees, _ := walk(t, l); len(trees) != 200 {
		t.Errorf("read %d trees, want 200", len(trees))
	}
}
