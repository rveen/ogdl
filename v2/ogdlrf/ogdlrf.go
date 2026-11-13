// Package ogdlrf implements OGDL remote functions: a request tree sent over
// a TCP connection, answered by a response tree (rfc/ogdl-binary-rfc.md,
// section 9).
//
// Two framings exist. Protocol 1 sends each message as a bare binary OGDL
// stream, delimited by its own end. Protocol 2, the default, precedes each
// stream by its length in four octets, most significant first.
//
// The first child of a request names the function. A Server with no route of
// that name answers "error notFound".
package ogdlrf

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/rveen/ogdl/v2"
)

// Function handles one request. The connection is given for information
// (the peer's address); the function must not read from or write to it.
type Function func(net.Conn, *ogdl.Node) *ogdl.Node

// DefaultMaxMessage bounds a protocol 2 message (RFC 11) unless MaxMessage
// is set.
const DefaultMaxMessage = 64 << 20

var (
	// ErrEmptyResponse is returned for a response tree with no nodes.
	ErrEmptyResponse = errors.New("ogdlrf: empty response")
	// ErrTooLarge is returned for a protocol 2 message above the maximum.
	ErrTooLarge = errors.New("ogdlrf: message too large")
)

// notFound is the response to a request for an unknown function.
func notFound() *ogdl.Node {
	return &ogdl.Node{Out: []*ogdl.Node{{Data: "error", Out: []*ogdl.Node{{Data: "notFound"}}}}}
}

// conn is one framed connection.
type conn struct {
	c        net.Conn
	r        *bufio.Reader
	protocol int
	max      int
}

func newConn(c net.Conn, protocol, max int) *conn {
	if max <= 0 {
		max = DefaultMaxMessage
	}
	return &conn{c: c, r: bufio.NewReader(c), protocol: protocol, max: max}
}

func (k *conn) write(n *ogdl.Node) error {
	b, err := n.MarshalBinary()
	if err != nil {
		return err
	}
	if k.protocol != 1 {
		if len(b) > k.max {
			return ErrTooLarge
		}
		var h [4]byte
		binary.BigEndian.PutUint32(h[:], uint32(len(b)))
		b = append(h[:], b...)
	}
	_, err = k.c.Write(b)
	return err
}

// read reads one message; io.EOF means the peer closed between messages.
func (k *conn) read() (*ogdl.Node, error) {
	if k.protocol == 1 {
		return ogdl.ReadBinary(k.r)
	}
	var h [4]byte
	if _, err := io.ReadFull(k.r, h[:]); err != nil {
		return nil, err // io.EOF if nothing was read
	}
	l := binary.BigEndian.Uint32(h[:])
	if uint64(l) > uint64(k.max) {
		return nil, ErrTooLarge
	}
	b := make([]byte, l)
	if _, err := io.ReadFull(k.r, b); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return ogdl.ParseBinary(b)
}

// Client calls the functions of one server, over one connection that it
// opens on first use and reopens when it fails. A Client is safe for
// concurrent use; calls are made one at a time.
type Client struct {
	Host       string        // host:port
	Timeout    time.Duration // per call; zero means none
	Protocol   int           // 1, or 2 (the default)
	MaxMessage int           // protocol 2; zero means DefaultMaxMessage

	mu sync.Mutex
	k  *conn
}

// Call sends req and returns the response. A call on a connection kept from
// an earlier call that fails is retried once on a new connection, since the
// server may have closed the old one; a function that must not run twice
// should not rely on a kept connection.
func (c *Client) Call(req *ogdl.Node) (*ogdl.Node, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	kept := c.k != nil
	resp, err := c.call(req)
	if err != nil && kept && c.k == nil {
		// The kept connection failed (and was dropped), not the call.
		resp, err = c.call(req)
	}
	return resp, err
}

func (c *Client) call(req *ogdl.Node) (*ogdl.Node, error) {
	if c.k == nil {
		nc, err := net.DialTimeout("tcp", c.Host, c.dialTimeout())
		if err != nil {
			return nil, fmt.Errorf("ogdlrf: cannot connect to %s: %w", c.Host, err)
		}
		c.k = newConn(nc, c.Protocol, c.MaxMessage)
	}
	if c.Timeout > 0 {
		c.k.c.SetDeadline(time.Now().Add(c.Timeout))
	}

	if err := c.k.write(req); err != nil {
		c.drop()
		return nil, err
	}
	resp, err := c.k.read()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		c.drop()
		return nil, err
	}
	if len(resp.Out) == 0 {
		return nil, ErrEmptyResponse
	}
	return resp, nil
}

func (c *Client) dialTimeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 30 * time.Second
}

func (c *Client) drop() {
	if c.k != nil {
		c.k.c.Close()
		c.k = nil
	}
}

// Close closes the client's connection, if any.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.drop()
	return nil
}

// Server serves functions by name.
type Server struct {
	Host       string        // host:port to listen on, for Serve
	Timeout    time.Duration // how long a connection may wait for its next request; zero means no limit
	Protocol   int           // 1, or 2 (the default)
	MaxMessage int           // protocol 2; zero means DefaultMaxMessage

	mu     sync.Mutex
	routes map[string]Function
	ln     net.Listener
	closed bool
}

// AddRoute serves f for requests whose first child is name.
func (s *Server) AddRoute(name string, f Function) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.routes == nil {
		s.routes = map[string]Function{}
	}
	s.routes[name] = f
}

// Serve listens on s.Host and serves until Close.
func (s *Server) Serve() error {
	ln, err := net.Listen("tcp", s.Host)
	if err != nil {
		return err
	}
	return s.ServeListener(ln)
}

// ServeListener serves the connections of ln until Close, after which it
// returns nil.
func (s *Server) ServeListener(ln net.Listener) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		ln.Close()
		return nil
	}
	s.ln = ln
	s.mu.Unlock()

	for {
		nc, err := ln.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return nil
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return err
		}
		go s.serveConn(nc)
	}
}

// Close stops accepting connections. Requests in progress finish.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func (s *Server) serveConn(nc net.Conn) {
	defer nc.Close()
	k := newConn(nc, s.Protocol, s.MaxMessage)
	for {
		if s.Timeout > 0 {
			nc.SetReadDeadline(time.Now().Add(s.Timeout))
		}
		req, err := k.read()
		if err != nil {
			if err != io.EOF {
				log.Printf("ogdlrf: %s: %v", nc.RemoteAddr(), err)
			}
			return
		}
		nc.SetReadDeadline(time.Time{})

		resp := s.handle(nc, req)
		if err := k.write(resp); err != nil {
			log.Printf("ogdlrf: %s: %v", nc.RemoteAddr(), err)
			return
		}
	}
}

func (s *Server) handle(nc net.Conn, req *ogdl.Node) (resp *ogdl.Node) {
	if len(req.Out) == 0 {
		return notFound()
	}
	s.mu.Lock()
	f := s.routes[req.Out[0].ThisString()]
	s.mu.Unlock()
	if f == nil {
		return notFound()
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ogdlrf: function %q panicked: %v", req.Out[0].ThisString(), r)
			resp = &ogdl.Node{Out: []*ogdl.Node{{Data: "error", Out: []*ogdl.Node{{Data: "panic"}}}}}
		}
	}()
	resp = f(nc, req)
	if resp == nil {
		resp = &ogdl.Node{}
	}
	return resp
}
