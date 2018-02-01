// Copyright 2017, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdlrf

import (
	"encoding/binary"
	"errors"
	"log"
	"net"
	"time"

	"github.com/rveen/ogdl"
)

var (
	errEmptyResponse = errors.New("Empty response")
	errWritingHeader = errors.New("error writing LEN header")
	errWritingBody   = errors.New("error writing body")
	errWriting       = errors.New("could not write all bytes")
)

// Client represents a the client side of a remote function (also known as a remote
// procedure call).
type Client struct {
	Host     string
	conn     net.Conn
	Timeout  int
	Protocol int
}

func (rf *Client) Dial() error {
	rf.Close()
	conn, err := net.Dial("tcp", rf.Host)
	rf.conn = conn
	return err
}

func (rf *Client) Call(g *ogdl.Graph) (*ogdl.Graph, error) {

	log.Printf("Client.Call to %s, %d", rf.Host, rf.Protocol)

	var err error
	var r *ogdl.Graph

	n := 2
	for {
		if rf.conn == nil {
			err = rf.Dial()
			if err != nil {
				return nil, errors.New("Cannot establish a connection to " + rf.Host)
			}
		}

		if rf.Protocol != 2 {
			r, err = rf.callV1(g)
		} else {
			r, err = rf.callV2(g)
		}
		if err == nil {
			break
		}
		n--
		if n < 0 {
			break
		}
		rf.conn = nil
		log.Println("Call.retry", n)
	}

	return r, err
}

// Call makes a remote call. It sends the given Graph in binary format to the server
// and returns the response Graph.
func (rf *Client) callV2(g *ogdl.Graph) (*ogdl.Graph, error) {

	// Convert graph to []byte
	buf := g.Binary()

	// Send LEN
	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, uint32(len(buf)))

	rf.conn.SetDeadline(time.Now().Add(time.Second * time.Duration(rf.Timeout)))
	i, err := rf.conn.Write(b4)
	if i != 4 || err != nil {
		log.Println("ogdlrf.Client, error writing LEN header", i, err)
		return nil, errWritingHeader
	}

	i, err = rf.conn.Write(buf)
	if err != nil {
		log.Println("ogdlrf.Client, error writing body,", err)
		return nil, errWritingBody
	}
	if i != len(buf) {
		log.Println("ogdlrf.Client, error writing body, LEN is", i, "should be", len(buf))
		return nil, errWritingBody
	}

	// Read header response
	j, err := rf.conn.Read(b4)
	if j != 4 {
		log.Println("error reading incomming message LEN")
		return nil, errors.New("error in message header")
	}
	l := binary.BigEndian.Uint32(b4)

	// Read body response
	buf3 := make([]byte, 0, l)
	tmp := make([]byte, 10000)
	l2 := uint32(0)
	log.Println("starting to read, should be", l)
	for {
		log.Println("reading ...")
		i, err = rf.conn.Read(tmp)
		log.Println("reading ...", i)
		l2 += uint32(i)
		if err != nil || i == 0 {
			log.Println("Error reading body", l2, l, err)
			return nil, err
		}

		buf3 = append(buf3, tmp[:i]...)

		if l2 >= l {
			break
		}
	}

	log.Println("read ...", len(buf3))
	g = ogdl.FromBinary(buf3)

	if g == nil || g.Len() == 0 {
		return nil, errEmptyResponse
	}

	// log.Println(" - end of Call")

	return g, err
}

func (rf *Client) callV1(g *ogdl.Graph) (*ogdl.Graph, error) {

	// log.Printf(" - callV1\n%s\n", g.Show())

	rf.conn.SetDeadline(time.Now().Add(time.Second * 10))

	b := g.Binary()
	n, err := rf.conn.Write(b)

	if err != nil {
		rf.conn = nil
		log.Println("callv1", err)
		return nil, err
	}
	if n != len(b) {
		rf.conn = nil
		log.Println("callv1", err)
		return nil, errWriting
	}

	// Read the incoming object
	g = ogdl.FromBinaryReader(rf.conn)

	if g == nil || g.Len() == 0 {
		return nil, errEmptyResponse
	}

	return g, nil
}

// Close closes the underlying connection, if open.
func (rf *Client) Close() {
	if rf.conn != nil {
		rf.conn.Close()
		rf.conn = nil
	}
}
