// (C) Copyright 2012-2013, Rolf Veen.
// See the LICENCE file.

package ogdl

// import "fmt"

type EventHandler interface {
    Event(string)
    EventL(string, int)
    Inc()
    Dec()
    Level() int
    SetLevel(int)
    Graph() * Graph
}

// EventHandlerG implements methods as per interface EventHandler. 
// This implementation builds a Graph from the events received.
//
type EventHandlerG struct {
    level int
    gl    []*Graph
}

// NewEventHandlerG creates an event handler
// that produces a Graph object from the 
// events sent.
//
func NewEventHandlerG() EventHandler {
    return &EventHandlerG{}
}

// Event creates a node at the current level
//
func (e *EventHandlerG) Event(s string) {

    // Create a transparent node to start with,
    // or else events at level 0 will overwrite
    // each other.
    if len(e.gl)==0 {
        e.gl = append(e.gl,NullGraph())
    }
    
    for len(e.gl) < e.level+2 {
        e.gl = append(e.gl,nil)
    }
    
    // Protection against holes can also be
    // done at other places in this package.
    if e.gl[e.level] == nil {
        println("event.go: e.gl[level-1 empty!")
        return
    }

    e.gl[e.level+1] = e.gl[e.level].Add(s)  
}

// EventL creates a node at the specified level
//
func (e *EventHandlerG) EventL(s string, l int) {
    e.level = l - 1;
    e.Event(s)
}

func (e *EventHandlerG) Level() int {
    return e.level
}

func (e *EventHandlerG) SetLevel(l int) {
    e.level = l
}

func (e *EventHandlerG) Inc() {
    e.level++
}

func (e *EventHandlerG) Dec() {
    if (e.level>0) {
        e.level--
    }
}

// Graph() returns the Graph object built from
// the events sent to this event handler.
//
func (e *EventHandlerG) Graph() *Graph {

    // It could happen that Graph() is requested
    // while no event has been sent, and thus
    // e.gl hasn't been initialized yet.
    if len(e.gl)==0 {
        return nil
    }
    
    // If the root node has only one subnode,
    // return that instead.
    if len(e.gl[0].Nodes)==1 {
        return e.gl[0].Nodes[0]
    }
    
    return e.gl[0]
}
