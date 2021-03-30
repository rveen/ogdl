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
	Host     string
	Timeout  int
	rtable   map[string]Function
	Protocol int
}

var notFound = ogdl.FromString("error notFound")

// AddRoute associates a handler function with the given path. A path in this
// context is the first child of the incomming request.
func (srv *Server) AddRoute(path string, f Function) {
	if srv.rtable == nil {
		srv.rtable = make(map[string]Function)
	}
	srv.rtable[path] = f
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
		}
		return notFound
	}
}

// Serve starts a remote function server. Handler functions should be set up with
// AddRoute.
func (srv *Server) Serve() error {
	if srv.Protocol == 1 {
		return Serve1(srv.Host, srv.router(), srv.Timeout)
	}
	return Serve(srv.Host, srv.router(), srv.Timeout)
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

	for {

		// Each message has a 4 byte header, an integer indicating the length:
		//
		//    message = LEN(uint32) BYTES
		//
		// Thus, first read 4 bytes (LEN)

		c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
		i, err := c.Read(b4)

		if i == 0 {
			break
		}

		if err != nil || i != 4 {
			log.Println("ogdlrf.Serve, error while trying to read LEN,", i, err)
			break
		}

		l := int(binary.BigEndian.Uint32(b4))
		log.Println("ogdlrf.Serve, rec LEN", l)
		if l == 0 {
			log.Println("LEN is 0, timeout was", timeout)
			break
		}

		// Read the body of the message

		buf := make([]byte, l)
		if buf == nil {
			log.Println("ogdlrf.Serve, cannot allocate memory for message")
			break
		}
		i, err = c.Read(buf)

		if err != nil {
			log.Println("ogdlrf.Serve, error reading message body, ", err)
			break
		}

		if i != l {
			log.Println("ogdlrf.Serve, error reading message body, LEN is", i, "should be", l)
			break
		}

		g := ogdl.FromBinary(buf)
		if g == nil || g.Out == nil {
			log.Println("ogdlrf.Serve, nothing in buf to produce a graph")
			break
		}
		r := handler(c, g)

		// Write message back
		buf = r.Binary()
		binary.BigEndian.PutUint32(b4, uint32(len(buf)))

		i, err = c.Write(b4)

		if i != 4 || err != nil {
			log.Println("ogdlrf.Serve, error writing LEN header", i, err)
			break
		}

		i, err = c.Write(buf)

		if err != nil {
			log.Println("ogdlrf.Serve, error writing body,", err)
			break
		}
		if i != len(buf) {
			log.Println("ogdlrf.Serve, error writing body, LEN is", i, "should be", len(buf))
			break
		}
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
			break
		}

		r := handler(c, g)

		// Write result in binary format
		b := r.Binary()
		i, err := c.Write(b)

		if err != nil {
			log.Println("ogdlrf.Serve, error writing body,", err)
			break
		}
		if i != len(b) {
			log.Println("ogdlrf.Serve, error writing body, LEN is", i, "should be", len(b))
		}
	}

	log.Println("ending Server.process and closing connection")
}
