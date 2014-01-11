// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"log"
	"net"
)

type RFunction struct {
	cfg  *Graph
	host string
	port string
	conn *net.TCPConn
}

func NewRFunction(g *Graph) *RFunction {
	rf := &RFunction{}
	rf.cfg = g
	rf.init()
	return rf
}

func (rf *RFunction) init() {

	rf.cfg.This = nil // XXX

	rf.host, _ = rf.cfg.GetString("host")
	rf.port, _ = rf.cfg.GetString("port")

	tcpAddr, err := net.ResolveTCPAddr("tcp", rf.host+":"+rf.port)
	if err != nil {
		log.Println("ResolveTCPAddr failed:", err.Error())
		return
	}

	rf.conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Println("Dial failed:", err.Error())
	}

	// Remote initialization
	r := rf.cfg.Node("init")
	if r != nil {
		rf.Call(r)
	}
}

func (rf *RFunction) Close() {

	// Remote close
	rf.Call(NewGraph("close"))

	// Local close
	if rf.conn != nil {
		rf.conn.Close()
	}
}

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

func (rf *RFunction) Call(g *Graph) (*Graph, error) {

	b := g.Binary()
	if b == nil {
		return nil, nil
	}

	log.Printf("Call(%s,%s)", rf.host, rf.port)

	// XXX also check if server side is alive

	if rf.conn == nil {
		log.Println("RFunction.Call: conn == nil !")
		rf.init()
		if rf.conn == nil {
			log.Println("RFunction.Call: conn == nil (2) !")
			return nil, nil
		}
	}

	_, err := rf.conn.Write(b)

	if err != nil {
		log.Println("RFunction.Call: write failed:", err.Error())
		rf.init()
		if rf.conn == nil {
			log.Println("RFunction.Call: conn == nil (3) !")
			return nil, nil
		}
		_, err = rf.conn.Write(b)
		if err != nil {
			return nil, nil
		}
	}

	p := NewBinParser(rf.conn)

	if p.read() != 0 {
		p.unread()
	}
	return p.Parse(), nil
}
