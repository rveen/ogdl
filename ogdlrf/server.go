// Copyright 2017, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdlrf

import (
	"encoding/binary"
	"log"
	"net"
	"time"

	"github.com/rveen/ogdl"
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

		if len(g.Out) == 0 {
			return notFound
		}

		s := g.Out[0].ThisString()

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
	defer l.Close()

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

// Serve1 starts a remote function server. Incomming requests should be handled
// by the given Function. This version of Serve doesn't work with AddRoute.
func Serve1(host string, handler Function, timeout int) error {

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
		go process1(conn, handler, timeout)
	}
}

// process handles the request through the given handler.
func process(c net.Conn, handler Function, timeout int) {

	defer c.Close()

	b4 := make([]byte, 4)

	log.Println("connection accepted")

	for {

		// Each message has a 4 byte header, an integer indicating the length:
		//
		//    message = LEN(uint32) BYTES
		//
		// Thus, first read 4 bytes (LEN)

		c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
		i, err := c.Read(b4)

		if i == 0 {
			return
		}

		if err != nil || i != 4 {
			log.Println("ogdlrf.Serve, error while trying to read LEN,", i, err)
			return
		}

		l := int(binary.BigEndian.Uint32(b4))
		log.Println("ogdlrf.Serve, rec LEN", l)

		// Read the body of the message

		buf := make([]byte, l)
		if buf == nil {
			log.Println("ogdlrf.Serve, cannot allocate memory for message")
			return
		}

		// Set a time out (maximum time for receiving the body)
		c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
		i, err = c.Read(buf)

		if err != nil {
			log.Println("ogdlrf.Serve, error reading message body, ", err)
			return
		}

		if i != l {
			log.Println("ogdlrf.Serve, error reading message body, LEN is", i, "should be", l)
			return
		}

		g := ogdl.FromBinary(buf)
		r := handler(c, g)

		// Write message back
		buf = r.Binary()
		binary.BigEndian.PutUint32(b4, uint32(len(buf)))

		i, err = c.Write(b4)

		if i != 4 || err != nil {
			log.Println("ogdlrf.Serve, error writing LEN header", i, err)
			return
		}

		i, err = c.Write(buf)

		if err != nil {
			log.Println("ogdlrf.Serve, error writing body,", err)
			return
		}
		if i != len(buf) {
			log.Println("ogdlrf.Serve, error writing body, LEN is", i, "should be", len(buf))
			return
		}
		log.Println("ogdlrf.Serve, body LEN", len(buf))
	}
}

// Old format, without the initial length indicator
func process1(c net.Conn, handler Function, timeout int) {

	defer c.Close()

	for {
		// Set a time out (maximum time until next message)
		c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))

		// Read the incoming object
		g := ogdl.FromBinaryReader(c)

		if g == nil {
			return
		}

		r := handler(c, g)

		// Write result in binary format
		b := r.Binary()
		i, err := c.Write(b)

		if err != nil {
			log.Println("ogdlrf.Serve, error writing body,", err)
			return
		}
		if i != len(b) {
			log.Println("ogdlrf.Serve, error writing body, LEN is", i, "should be", len(b))
		}
	}
}
