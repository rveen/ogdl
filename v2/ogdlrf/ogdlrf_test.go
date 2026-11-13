package ogdlrf

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rveen/ogdl/v2"
)

func tree(s string) *ogdl.Node {
	n, err := ogdl.ParseString(s)
	if err != nil {
		panic(err)
	}
	return n
}

// start runs a server with an "echo" route (answers "echo" and the request's
// second child) and returns its address.
func start(t *testing.T, protocol int) (string, *Server, *int64) {
	t.Helper()
	var calls int64
	s := &Server{Protocol: protocol}
	s.AddRoute("echo", func(_ net.Conn, req *ogdl.Node) *ogdl.Node {
		atomic.AddInt64(&calls, 1)
		return &ogdl.Node{Out: []*ogdl.Node{{Data: "echo", Out: []*ogdl.Node{req.N(1)}}}}
	})
	s.AddRoute("empty", func(net.Conn, *ogdl.Node) *ogdl.Node {
		atomic.AddInt64(&calls, 1)
		return nil
	})
	s.AddRoute("panic", func(net.Conn, *ogdl.Node) *ogdl.Node { panic("boom") })
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot listen:", err)
	}
	go s.ServeListener(ln)
	t.Cleanup(func() { s.Close() })
	return ln.Addr().String(), s, &calls
}

func TestCall(t *testing.T) {
	for _, p := range []int{1, 2} {
		addr, _, _ := start(t, p)
		c := &Client{Host: addr, Protocol: p, Timeout: 5 * time.Second}
		defer c.Close()

		for i := 0; i < 3; i++ { // the connection is kept between calls
			resp, err := c.Call(tree("echo\nhello"))
			if err != nil || resp.N(0).ThisString() != "echo" || resp.N(0).N(0).ThisString() != "hello" {
				t.Fatalf("protocol %d: %v, %v", p, resp, err)
			}
		}

		resp, err := c.Call(tree("nosuch"))
		if err != nil || resp.Text() != "error\n  notFound\n" {
			t.Errorf("protocol %d, unknown function: %q, %v", p, resp.Text(), err)
		}
		resp, err = c.Call(tree("panic"))
		if err != nil || resp.N(0).ThisString() != "error" {
			t.Errorf("protocol %d, panic: %v, %v", p, resp, err)
		}
		// Binary data passes through.
		req := &ogdl.Node{Out: []*ogdl.Node{{Data: "echo"}, {Data: []byte{0, 1, 2}}}}
		resp, err = c.Call(req)
		if b, ok := resp.N(0).N(0).Data.([]byte); err != nil || !ok || len(b) != 3 {
			t.Errorf("protocol %d, data: %v, %v", p, resp, err)
		}
	}
}

// An empty response is an error, and the request is not sent again.
func TestEmptyResponseNotRetried(t *testing.T) {
	addr, _, calls := start(t, 2)
	c := &Client{Host: addr}
	defer c.Close()
	c.Call(tree("echo\nx")) // keep a connection
	atomic.StoreInt64(calls, 0)
	if _, err := c.Call(tree("empty")); !errors.Is(err, ErrEmptyResponse) {
		t.Errorf("got %v, want ErrEmptyResponse", err)
	}
	if n := atomic.LoadInt64(calls); n != 1 {
		t.Errorf("the function ran %d times", n)
	}
}

// A message delivered in pieces is read whole (the reference implementation
// read the body with a single Read).
func TestSplitFrame(t *testing.T) {
	addr, _, _ := start(t, 2)
	nc, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	body, _ := tree("echo\nsplit").MarshalBinary()
	msg := binary.BigEndian.AppendUint32(nil, uint32(len(body)))
	msg = append(msg, body...)
	for _, b := range msg {
		nc.Write([]byte{b})
		time.Sleep(time.Millisecond)
	}
	nc.SetDeadline(time.Now().Add(5 * time.Second))
	var h [4]byte
	if _, err := io.ReadFull(nc, h[:]); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, binary.BigEndian.Uint32(h[:]))
	if _, err := io.ReadFull(nc, b); err != nil {
		t.Fatal(err)
	}
	resp, err := ogdl.ParseBinary(b)
	if err != nil || resp.N(0).N(0).ThisString() != "split" {
		t.Errorf("%v, %v", resp, err)
	}
}

func TestTooLarge(t *testing.T) {
	addr, _, _ := start(t, 2) // the server uses DefaultMaxMessage
	nc, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	nc.Write(binary.BigEndian.AppendUint32(nil, DefaultMaxMessage+1))
	nc.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := nc.Read(make([]byte, 1)); err == nil {
		t.Error("the server answered an oversized message instead of closing")
	}

	c := &Client{Host: addr, MaxMessage: 10}
	defer c.Close()
	if _, err := c.Call(tree("echo\nrequest-longer-than-ten-octets")); !errors.Is(err, ErrTooLarge) {
		t.Errorf("client: got %v, want ErrTooLarge", err)
	}
}

// A kept connection the server closed is replaced, once.
func TestReconnect(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip(err)
	}
	s := &Server{}
	s.AddRoute("echo", func(_ net.Conn, req *ogdl.Node) *ogdl.Node { return tree("ok") })
	go s.ServeListener(ln)
	defer s.Close()

	c := &Client{Host: ln.Addr().String()}
	defer c.Close()
	if _, err := c.Call(tree("echo")); err != nil {
		t.Fatal(err)
	}
	c.k.c.Close() // as if the server had dropped it
	if resp, err := c.Call(tree("echo")); err != nil || resp.N(0).ThisString() != "ok" {
		t.Errorf("after the connection broke: %v, %v", resp, err)
	}
}

func TestConcurrentClient(t *testing.T) {
	addr, _, calls := start(t, 2)
	c := &Client{Host: addr}
	defer c.Close()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if resp, err := c.Call(tree("echo\nx")); err != nil || resp.N(0).N(0).ThisString() != "x" {
					t.Error(resp, err)
				}
			}
		}()
	}
	wg.Wait()
	if n := atomic.LoadInt64(calls); n != 200 {
		t.Errorf("%d calls served, want 200", n)
	}
}

func TestNoServer(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	c := &Client{Host: addr, Timeout: time.Second}
	if _, err := c.Call(tree("echo")); err == nil {
		t.Error("call without a server succeeded")
	}
}
