// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bytes"
	"os"
)

// Log is a log store for binary OGDL objects.
//
// All objects are appended to a file, and a position is returned.
//
type Log struct {
	f        *os.File
	autoSync bool
	b        bytes.Buffer
}

// OpenLog opens a log file. If the file doesn't exist, it is created.
func OpenLog(file string) (*Log, error) {

	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	log := Log{f: f, autoSync: true}

	return &log, nil
}

// Close closes a log file
func (log *Log) Close() {
	if log.f != nil {
		log.f.Close()
	}
}

// Sync commits the changes to disk (the exact behavior is OS dependent).
func (log *Log) Sync() {
	if log.f != nil {
		log.f.Sync()
	}
}

// Sync commits the changes to disk (the exact behavior is OS dependent).
func (log *Log) SetSync(sync bool) {
	log.autoSync = sync
}

// Add adds an OGDL object to the log. The starting position into the log
// is returned.
func (log *Log) Add(g *Graph) int64 {

	b := g.Binary()

	if b == nil {
		return 0
	}

	if log.f != nil {

		i, _ := log.f.Seek(0, 2)

		log.f.Write(b)

		if log.autoSync {
			log.f.Sync()
		}

		return i
	} else {

		i := log.b.Len()
		log.b.Write(b)

		return int64(i)
	}
}

// Bytes works only if we have been writing to the byte buffer, not to a file
func (log *Log) Bytes() []byte {
	if log.f != nil {
		return nil
	}
	return log.b.Bytes()
}

// AddBinary adds an OGDL binary object to the log. The starting position into
// the log is returned.
func (log *Log) AddBinary(b []byte) int64 {

	if log.f != nil {
		i, _ := log.f.Seek(0, 2)
		log.f.Write(b)

		if log.autoSync {
			log.f.Sync()
		}

		return i
	} else {

		i := log.b.Len()
		log.b.Write(b)

		return int64(i)
	}
}

// Get returns the OGDL object at the position given and the position of the
// next object, or an error.
func (log *Log) Get(i int64) (*Graph, int64, error) {

	/* Position in file */
	_, err := log.f.Seek(i, 0)
	if err != nil {
		return nil, -1, err
	}

	p := newBinParser(log.f)
	g := p.parse()

	if p.n == 0 {
		return g, -1, nil
	}

	return g, i + int64(p.n), err
}

// GetBinary returns the OGDL object at the position given and the position of the
// next object, or an error. The object returned is in binary form, exactly
// as it is stored in the log.
func (log *Log) GetBinary(i int64) ([]byte, int64, error) {

	// Position in file
	_, err := log.f.Seek(i, 0)
	if err != nil {
		return nil, 0, err
	}

	/* Read until EOS of binary OGDL.

	   There should be a Header first.
	*/
	p := newBinParser(log.f)

	if !p.header() {
		return nil, 0, err
	}
	for {
		lev, _, _ /* typ, b*/ := p.line(false)
		if lev == 0 {
			break
		}
	}

	n := p.n

	// Read bytes
	b := make([]byte, n)
	_, err = log.f.ReadAt(b, i)

	return b, int64(n), err
}
