// Copyright 2012-2015, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"encoding/binary"
	"errors"
	"log"
	"net"
	"time"
)

// RFunction represents a remote function (also known as a remote procedure
// call).
type RFunction struct {
	cfg      *Graph
	host     string
	conn     net.Conn
	protocol int
	valid    bool
}

// NewRFunction opens a connection to a TCP/IP server specified in the
// Graph supplied. It also makes an initialization call, if the Graph has an
// 'init' section.
func NewRFunction(cfg *Graph) (*RFunction, error) {

	host, _ := cfg.GetString("host")
	prot, _ := cfg.GetInt64("protocol")

	log.Println("rf: host", host, ", protocol", prot, cfg.Show())

	rf := &RFunction{}

	tcpAddr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	rf.cfg = cfg
	rf.host = host
	rf.conn = conn
	rf.protocol = int(prot)

	return rf, nil
}

func TCPRawServer(host string, handler func(c net.Conn, b []byte) []byte, timeout int) error {

	l, err := net.Listen("tcp", host)
	if err != nil {
		return err
	}

	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		// Handle the connection in a new goroutine.
		go rawhandler(conn, handler, timeout)
	}
}

func TCPServerV2(host string, handler func(net.Conn, *Graph) *Graph, timeout int) error {

	l, err := net.Listen("tcp", host)
	if err != nil {
		return err
	}

	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		// Handle the connection in a new goroutine.
		go handler_v2(conn, handler, timeout)
	}
}

func TCPServerV1(host string, handler func(net.Conn, *Graph) *Graph, timeout int) error {

	l, err := net.Listen("tcp", host)
	if err != nil {
		return err
	}

	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		// Handle the connection in a new goroutine.
		go handler_v1(conn, handler, timeout)
	}
}

func handler_v1(c net.Conn, handler func(net.Conn, *Graph) *Graph, timeout int) {

	for {
		// Set a time out (maximum time until next message)
		c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))

		// Read the incoming object
		p := newBinParser(c)
		g := p.Parse()

		if g == nil {
			c.Close()
			return
		}

		r := handler(c, g)

		// Write to result back in binary format
		b := r.Binary()
		c.Write(b)
	}
}

func handler_v2(c net.Conn, handler func(net.Conn, *Graph) *Graph, timeout int) {

	b4 := make([]byte, 4)

	log.Println("connection accepted")

	for {

		// Set a time out (maximum time until next message)
		c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))

		// Read message: LEN(uint32) BYTES

		// Read 4 bytes (LEN)
		i, err := c.Read(b4)
		// Set a time out (maximum time for receiving the body)
		c.SetReadDeadline(time.Now().Add(time.Millisecond * 1000))

		if err != nil || i != 4 {
			log.Println("connection closed", i, err)
			c.Close()
			return
		}

		l := int(binary.BigEndian.Uint32(b4))

		// read body of message
		buf := make([]byte, l)
		if buf == nil {
			log.Println("connection closed: not more memory")
			c.Close()
			return
		}
		i, err = c.Read(buf)

		if err != nil || i != l {
			log.Println("connection pre-closed", err)
			c.Close()
			return
		}

		g := FromBinary(buf)
		r := handler(c, g)

		// Write message back
		buf = r.Binary()
		binary.BigEndian.PutUint32(b4, uint32(len(buf)))

		c.Write(b4)
		c.Write(buf)
	}
}

func rawhandler(c net.Conn, handler func(c net.Conn, body []byte) []byte, timeout int) {

	b4 := make([]byte, 4)

	log.Println("connection accepted")

	for {

		// Set a time out (maximum time until next message)
		c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))

		// Read message: LEN(uint32) BYTES

		// Read 4 bytes (LEN)
		i, err := c.Read(b4)
		// Set a time out (maximum time for receiving the body)
		c.SetReadDeadline(time.Now().Add(time.Millisecond * 100))

		if err != nil || i != 4 {
			log.Println("connection closed", i, err)
			c.Close()
			return
		}

		l := int(binary.BigEndian.Uint32(b4))

		// read body of message
		buf := make([]byte, l)
		if buf == nil {
			log.Println("connection closed: not more memory")
			c.Close()
			return
		}
		i, err = c.Read(buf)
		if err != nil || i != l {
			log.Println("connection pre-closed", err)
			c.Close()
			return
		}

		r := handler(c, buf)

		// Write back

		l = len(r)
		binary.BigEndian.PutUint32(b4, uint32(l))
		c.Write(b4)
		c.Write(r)
	}
}

// Call makes a remote call. It sends the given Graph in binary format to the server
// and returns the response Graph.
func (rf *RFunction) Call(g *Graph) (*Graph, error) {

	var r *Graph
	var err error
	var addr *net.TCPAddr

	if rf.protocol == 2 {
		r, err = rf.callV2(g)
	} else {
		r, err = rf.callV1(g)
	}

	if err != nil {
		log.Println("Call failed. Retrying", err.Error())
		addr, err = net.ResolveTCPAddr("tcp", rf.host)
		if err != nil {
			log.Println("Call failed", err.Error())
			return nil, err
		}

		rf.conn, err = net.DialTCP("tcp", nil, addr)
		if err != nil {
			log.Println("Call failed", err.Error())
			return nil, err
		}

		if rf.protocol == 2 {
			r, err = rf.callV2(g)
		} else {
			r, err = rf.callV1(g)
		}
	}

	if err != nil {
		log.Println("Call failed", err.Error())
	}
	return r, err
}

func (rf *RFunction) callV2(g *Graph) (*Graph, error) {

	buf := g.Binary()
	buf2 := make([]byte, 4+len(buf))

	for i := 0; i < len(buf); i++ {
		buf2[i+4] = buf[i]
	}

	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(buf2, uint32(len(buf)))

	// Write request (len + body)
	rf.conn.Write(buf2)

	// Read header response
	j, _ := rf.conn.Read(b4)
	if j != 4 {
		rf.conn = nil
		log.Println("callv2 error, message header")
		return nil, errors.New("error in message header")
	}
	l := binary.BigEndian.Uint32(b4)

	// Read body response
	buf3 := make([]byte, l)
	j, err := rf.conn.Read(buf3)
	if err != nil {
		rf.conn = nil
		log.Println("callv2", err)
		return nil, err
	}
	if j != int(l) {
		rf.conn = nil
		log.Println("callv2 error, message len")
		return nil, errors.New("error in message len")
	}

	g = FromBinary(buf3)

	return g, nil
}

func (rf *RFunction) callV1(g *Graph) (*Graph, error) {

	b := g.Binary()
	n, err := rf.conn.Write(b)
	if err != nil {
		rf.conn = nil
		log.Println("callv1", err)
		return nil, err
	}
	if n < len(b) {
		rf.conn = nil
		log.Println("callv1", err)
		return nil, errors.New("could not write all bytes")
	}

	p := newBinParser(rf.conn)

	c := p.read()
	if c == -1 {
		rf.conn = nil
		log.Println("callv1", "EOS")
		return nil, errors.New("unexpected EOS")
	}

	p.unread()

	r := p.Parse()

	if r == nil || r.Len() == 0 {
		return nil, errors.New("nil response")
	}

	return r, nil
}

func (rf *RFunction) Close() {
	if rf.conn != nil {
		rf.conn.Close()
		rf.conn = nil
	}
}
