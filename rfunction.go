// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"errors"
	"net"
)

type RFunction struct {
	cfg  *Graph
	host string
	port string
	conn *net.TCPConn
}

// RFunctions opens a connection to a TCP/IP server specified in the
// Graph supplied. It also makes an initialization call, if the Graph has an
// 'init' section.
func NewRFunction(g *Graph) (*RFunction, error) {
	rf := &RFunction{}
	rf.cfg = g
	err := rf._init()
	return rf, err
}

// Do not want to collide with init()
func (rf *RFunction) _init() error {

	rf.cfg.This = nil // XXX

	rf.host, _ = rf.cfg.GetString("host")
	rf.port, _ = rf.cfg.GetString("port")

	addr := rf.host + ":" + rf.port
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	rf.conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	}

	// Remote initialization
	r := rf.cfg.Node("init")
	if r != nil {
		rf.Call(r)
	}
	return nil
}

// Close the connection to the remote server.
func (rf *RFunction) Close() {

	// Remote close
	rf.Call(NewGraph("close"))

	// Local close
	if rf.conn != nil {
		rf.conn.Close()
	}
}

// Send opens a connection to a remote server and makes a remote call. It sends
// the given Graph in binary format to the server and returns the response Graph.
// The connection is closed before leaving this method.
func (g *Graph) Send(cfg *Graph) (*Graph, error) {

	b := g.Binary()

	if b == nil {
		return nil, nil
	}

	host, _ := cfg.GetString("host")

	tcpAddr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write(b)
	if err != nil {
		return nil, err
	}

	p := NewBinParser(conn)
	n := p.Parse()

	return n, nil
}

// Call makes a remote call. It sends the given Graph in binary format to the server
// and returns the response Graph.
func (rf *RFunction) Call(g *Graph) (*Graph, error) {

	b := g.Binary()
	if b == nil {
		return nil, nil
	}

	// XXX also check if server side is alive

	if rf.conn == nil {
		rf._init()
		if rf.conn == nil {
			return nil, errors.New("No connection")
		}
	}

	_, err := rf.conn.Write(b)

	if err != nil {
		rf._init()
		if rf.conn == nil {
			return nil, errors.New("No connection")
		}
		_, err = rf.conn.Write(b)
		if err != nil {
			return nil, errors.New("Cannot write to connection")
		}
	}

	p := NewBinParser(rf.conn)

	if p.read() != 0 {
		p.unread()
	}
	return p.Parse(), nil
}
