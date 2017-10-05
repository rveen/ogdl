// Copyright 2017, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdlrf

import (
	"encoding/binary"
	"log"
	"net"
	"ogdl"
	"time"
)

// Server hold the state of the server side of a remote function.
type Server struct {
	Host    string
	Timeout int
	rtable  map[string]Function
}

var notFound = ogdl.FromString("error notFound")

// AddRoute associates a handler function with the given path. A path in this
// context is the first child of the incomming request.
func (s *Server) AddRoute(path string, f Function) {
	if s.rtable == nil {
		s.rtable = make(map[string]Function)
	}
	s.rtable[path] = f
}

func (srv *Server) router() Function {
	return func(c net.Conn, g *ogdl.Graph) *ogdl.Graph {
		s := g.Out[0].ThisString()
		log.Println(s)

		h := srv.rtable[s]
		if h != nil {
			return h(c, g)
		} else {
			return notFound
		}
	}
}

// Serve starts a remote function server. Handler functions should be set up with
// AddRoute.
func (s *Server) Serve() error {
	return Serve(s.Host, s.router(), s.Timeout)
}

// Serve starts a remote function server. Incomming requests should be handled
// by the given Function. This version of Serve doesn't work with AddRoute.
func Serve(host string, handler Function, timeout int) error {

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
		go process(conn, handler, timeout)
	}
}

// process handles the request through the given handler.
func process(c net.Conn, handler Function, timeout int) {

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
			log.Println("connection closed: no more memory")
			c.Close()
			return
		}
		i, err = c.Read(buf)

		if err != nil || i != l {
			log.Println("connection pre-closed", err)
			c.Close()
			return
		}

		g := ogdl.FromBinary(buf)
		r := handler(c, g)

		// Write message back
		buf = r.Binary()
		binary.BigEndian.PutUint32(b4, uint32(len(buf)))

		c.Write(b4)
		c.Write(buf)
	}
}
