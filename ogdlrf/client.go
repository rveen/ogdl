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

// Client represents a the client side of a remote function (also known as a remote
// procedure call).
type Client struct {
	Host    string
	conn    net.Conn
	Timeout int
}

// Dial connect this client to the specified server
func Dial(host string) (*Client, error) {
	client := &Client{Host: host, Timeout: 1}

	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	client.conn = conn
	return client, nil
}

// Call makes a remote call. It sends the given Graph in binary format to the server
// and returns the response Graph.
func (rf *Client) Call(g *ogdl.Graph) (*ogdl.Graph, error) {

	// open the connection, and make sure it is closed at the end
	conn, err := net.Dial("tcp", rf.Host)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Convert graph to []byte
	buf := g.Binary()

	// Send LEN
	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, uint32(len(buf)))

	conn.SetDeadline(time.Now().Add(time.Second * 10))
	i, err := conn.Write(b4)
	if i != 4 || err != nil {
		log.Println("ogdlrf.Serve, error writing LEN header", i, err)
		return nil, errors.New("error writing LEN header")
	}

	i, err = conn.Write(buf)
	if err != nil {
		log.Println("ogdlrf.Serve, error writing body,", err)
		return nil, errors.New("error writing body")
	}
	if i != len(buf) {
		log.Println("ogdlrf.Serve, error writing body, LEN is", i, "should be", len(buf))
		return nil, errors.New("error writing body")
	}

	// Read header response
	conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	j, err := conn.Read(b4)
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
		i, err = conn.Read(tmp)
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

	// log.Println(" - end of Call")

	return g, err
}

// Call makes a remote call. It sends the given Graph in binary format to the server
// and returns the response Graph.
func (rf *Client) Call1(g *ogdl.Graph) (*ogdl.Graph, error) {

	nretry := 2

	// log.Printf("Client.Call\n%s", g.Show())

retry:
	if rf.conn == nil {
		conn, err := net.Dial("tcp", rf.Host)
		if err != nil {
			rf.conn = nil
			return nil, err
		}
		rf.conn = conn
	}

	buf := g.Binary()

	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, uint32(len(buf)))

	//log.Println("rf.Call, wr len ", len(buf))

	// Write request (len + body)
	rf.conn.SetDeadline(time.Now().Add(time.Second * 10))
	i, err := rf.conn.Write(b4)
	if i != 4 || err != nil {
		log.Println("ogdlrf.Serve, error writing LEN header", i, err)
		return nil, errors.New("error writing LEN header")
	}

	i, err = rf.conn.Write(buf)
	if err != nil {
		log.Println("ogdlrf.Serve, error writing body,", err)
		return nil, errors.New("error writing body")
	}
	if i != len(buf) {
		log.Println("ogdlrf.Serve, error writing body, LEN is", i, "should be", len(buf))
		return nil, errors.New("error writing body")
	}

	// Read header response
	rf.conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	j, err := rf.conn.Read(b4)
	if j != 4 {
		rf.Close()
		rf.conn = nil
		nretry--
		if nretry > 0 {
			goto retry
		}
		log.Println("error reading incomming message LEN")
		return nil, errors.New("error in message header")
	}
	l := binary.BigEndian.Uint32(b4)

	// Read body response
	buf3 := make([]byte, l)
	j, err = rf.conn.Read(buf3)
	if err != nil {
		rf.conn = nil
		log.Println(err)
		return nil, err
	}
	if j != int(l) {
		rf.conn = nil
		log.Println("error in message len", i, l)
		return nil, errors.New("error in message len")
	}

	g = ogdl.FromBinary(buf3)

	// log.Println(" - end of Call")

	return g, err
}

// Close closes the underlying connection, if open.
func (rf *Client) Close() {
	if rf.conn != nil {
		rf.conn.Close()
		rf.conn = nil
	}
}
