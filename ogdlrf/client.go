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
	Host string
	conn net.Conn
}

// Dial connect this client to the specified server
func Dial(host string) (*Client, error) {
	client := &Client{Host: host}

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

	var err error
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
	rf.conn.Write(b4)
	rf.conn.Write(buf)

	// Read header response
	rf.conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	j, _ := rf.conn.Read(b4)
	if j != 4 {
		rf.Close()
		rf.conn = nil
		nretry--
		if nretry > 0 {
			goto retry
		}
		log.Println("error in message header")
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
		log.Println("error in message len")
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
