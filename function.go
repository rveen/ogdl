// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"log"
	"reflect"
)

// Types for OGDL templates.
//
// instantiate -> init -> remote init -> exec
//
// $store.get(1), a remote function. Needs to open a database. Needs to be
// closed.
//
// Configuration:
//
//   store
//     !type
//       ogdl.RFunction
//     !init
//        host localhost
//        port 1122
//        remote
//          database db1
//          user $user
//
// $math.sin(1.0), a local function with one numeric argument
//
// $math.max(1,0)
//
// Input
//
//   math
//     max
//       !g
//          1
//          0

// factory is a map that associates names with functions
// that instantiate objects (types).
var factory map[string]func() interface{}

var functions map[string]func(g *Graph, p *Graph, i int) []byte // interface{}

func FunctionAddToFactory(s string, f func() interface{}) {
	factory[s] = f
}

func FunctionAdd(s string, f func(*Graph, *Graph, int) []byte) {
	functions[s] = f
}

func (g *Graph) Function(p *Graph, ix int, context *Graph) (interface{}, error) {

	n := g.Node("!type")

	if n == nil || n.Len() == 0 {
		return nil, nil
	}

	name := n.GetAt(0).String()

	log.Println("Function(): !type ", name)

	// Case 1: simple function
	//
	// If type == "function", then call a function directly from the
	// functions[] table, no need to instantiate an object.

	if "function" == name {

		funame := p.GetAt(ix - 1).String()

		log.Println("Function(): ", funame)

		fu := functions[funame]
		if fu == nil {
			return nil, nil
		}

		arg := NilGraph()
		args := p.Out[ix]

		for i := 0; i < args.Len(); i++ {
			v := context.Eval(args.Out[i])

			arg.Add(_string(v))
		}
		// log.Printf("Function().Call:\n%s\n", arg.Text())

		return fu(context, arg, 0), nil
	}

	// Case 2: remote function

	if "rfunction" == name {

		var rf *RFunction

		if n.Len() == 1 {
			rf = NewRFunction(g.Node("!init"))
			n.Add(rf)
		} else {
			rf = n.GetAt(1).This.(*RFunction)
		}

		arg := NewGraph(p.Out[ix].String())
		args := p.Out[ix+1]

		for _, a := range args.Out {
			v := context.Eval(a)

			if _, ok := v.(*Graph); ok {
				arg.Add("_").Add(v)
			} else {
				arg.Add(v)
			}
		}
		log.Printf("RFunction().Call:\n%s\n", arg.Text())
		return rf.Call(arg)
	}

	// Case 3: object with methods to be discovered through reflection

	var v reflect.Value

	// If !type has a second node, that means that it has been instantiated
	// already. The second node points to the type's instance.

	if n.Len() == 1 {

		// !type has one node, so instantiate.

		ff := factory[name]
		if ff == nil {
			return nil, nil
		}

		//f := reflect.ValueOf(ff)

		//v = f.Call(nil)[0]
		itf := ff()
		v = reflect.ValueOf(itf)

		//v = v.Elem() // works because we have defined interface{} as return

		// Add the object as second node of !type. Next time w'll pick this
		// object.

		n.Add(v)

		// If !init is defined, the init(Graph) function is called on the
		// instantiated type.

		nn := g.Node("!init")

		if nn != nil {
			v.MethodByName("Init").Call([]reflect.Value{reflect.ValueOf(nn)})
		}
	} else {
		v = n.GetAt(1).This.(reflect.Value)
	}

	// exec: as per path
	//
	// p[i] is function or field name
	// rest are arguments.

	// obj = p.GetAt(i)

	fn := p.GetAt(ix)
	ag := p.GetAt(ix + 1)

	if "!g" != ag.String() {
		return nil, nil // XXX
	}

	log.Println("function name", fn)

	if fn == nil {
		return "No method " + fn.Text(), nil //XXX
	}

	fname := fn.String()

	// Predefined functions: destroy

	// Check if it is a field

	// Check if it is a method
	me := v.MethodByName(fname)

	if !me.IsValid() {
		return "(2) No method " + fname, nil // XXX
	}

	// Build arguments in the form []reflect.Value

	var args []reflect.Value

	for _, arg := range ag.Out {
		a := g.Eval(arg)
		log.Println("- arg", reflect.TypeOf(a).String())
		args = append(args, reflect.ValueOf(a))
	}

	return me.Call(args)[0].Interface(), nil
}

func init() {
	factory = make(map[string]func() interface{})
	factory["nil"] = nilGraphI

	functions = make(map[string]func(g *Graph, p *Graph, i int) []byte)
	functions["T"] = templateProcess
}

// Example functions and objects

func templateProcess(context *Graph, p *Graph, i int) []byte {
	t := NewTemplate(p.Text())
	return t.Process(context)
}

func nilGraphI() interface{} {
	return NilGraph()
}
