// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import "os"

// Log is a log store for binary OGDL objects.
//
// All objects are appended to a file, and a position is returned.
//
type Log struct {
	f        *os.File
	autoSync bool
}

func OpenLog(file string) (*Log, error) {

	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	log := Log{f, true}

	return &log, nil
}

func (log *Log) Close() {
	log.f.Close()
}

func (log *Log) Sync() {
	log.f.Sync()
}

func (log *Log) Add(g *Graph) int64 {

	b := g.Binary()

	if b == nil {
		return 0
	}

	log.f.Write(b)

	i, _ := log.f.Seek(0, 2)
	if log.autoSync {
		log.f.Sync()
	}

	return i
}

func (log *Log) AddBinary(b []byte) int64 {

	log.f.Write(b)
	i, _ := log.f.Seek(0, 2)

	if log.autoSync {
		log.f.Sync()
	}

	return i
}

// Get returns the object at the position given,
// an eventual error, and the position of the
// next object.
func (log *Log) Get(i int64) (*Graph, error, int64) {

	/* Position in file */
	_, err := log.f.Seek(i, 0)
	if err != nil {
		return nil, err, -1
	}

	p := NewBinParser(log.f)
	g := p.Parse()

	return g, err, i + int64(p.n)
}

func (log *Log) GetBinary(i int64) ([]byte, error, int64) {

	// Position in file
	_, err := log.f.Seek(i, 0)
	if err != nil {
		return nil, err, 0
	}

	/* Read until EOS of binary OGDL.

	   There should be a Header first.
	*/
	p := NewBinParser(log.f)

	if !p.header() {
		return nil, err, 0
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

	return b, err, int64(n)
}
