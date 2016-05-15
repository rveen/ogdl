// Copyright 2012-2015, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"errors"
	"log"
	"reflect"
)

// factory[] is a map that stores type constructors.
var factory map[string]func() interface{}

// functions[] is a map for storing functions with a suitable signature so that
// they can be called from within templates.
var functions map[string]func(*Graph, []interface{}) []byte // interface{}

// FunctionAddConstructor adds a factory kind of function to the context.
func FunctionAddConstructor(s string, f func() interface{}) {
	factory[s] = f
}

// FunctionAdd adds a function to the context.
func FunctionAdd(s string, f func(*Graph, []interface{}) []byte) {
	functions[s] = f
}

// Function enables calling Go functions from templates. Paths in templates
// are translated into Go functions if !type definitions are present.
//
// Functions and type methods are handled here, based on the two maps, factory[]
// and functions[].
//
// BUG Detect methods directly from the interface
//
// Also remote functions are called from here. A remote function is a call to
// a TCP/IP server, in which both the request and the response are binary encoded
// OGDL objects.
//
// (This code can be much improved)
//
// INPUT FORMAT
//
// g is the Function's context. g.This contains the presumed class name.
// The _type subnode of g, if present, contains the function type (a Go
// interface name or 'rfunction'
//
// p is the input path, where i points to the current position to be processed.
// The arguments of the function are 1 level higher than the function name.
// p[ix] points to the class name.
//
// Example
//
// !p
//   T
//   !g
//     'some text'
//
// EXAMPLE 2
// !p
//   math
//   Sin
//   !g
//    !e
//      1.0
//
func (g *Graph) Function(p *Graph, ix int, context *Graph) (interface{}, error) {

	// !type may be there. If not, then a simple function is assumed.
	n := g.Node("!type")

	if n == nil || n.Len() == 0 {

		// Case 1: simple function
		funame := p.GetAt(ix).ThisString()
		fu := functions[funame]

		if fu == nil {
			return nil, errors.New("function not in table " + funame)
		}

		// Prepare a pre-evaluated parameter array
		var args []interface{}
		for _, arg := range p.Out[ix+1].Out {
			args = append(args, context.EvalExpression(arg))
		}
		return fu(context, args), nil
		// return nil, nil
	}

	name := n.String()

	// Case 2: remote function

	if "rfunction" == name {

		var rf *RFunction
		var err error

		if n.Len() == 1 {
			rf, err = NewRFunction(g.Node("!init"))
			if err != nil {
				return nil, err
			}
			n.Add(rf)
		} else {
			rf = n.GetAt(1).This.(*RFunction)
			if rf.conn == nil {
				rf, err = NewRFunction(g.Node("!init"))
				if err != nil {
					return nil, err
				}
				n.DeleteAt(1)
				n.Add(rf)
			}
		}

		args := New()
		par := args.Add(p.Out[ix].This)
		par.Copy(p.Out[ix+1])

		log.Printf("rfuntion args(pre) \n%s\n", args.Show())
		context.evalGraph(par)
		log.Printf("rfunction args\n%s\n", args.Show())

		return rf.Call(args)
	}

	// Case 3: object with methods to be discovered through reflection

	var v reflect.Value

	// If !type has a second node, that means that it has been instantiated
	// already. The second node points to the type's instance.

	if n.Len() == 1 {

		// !type has one node, so instantiate.

		ff := factory[name]
		if ff == nil {
			log.Printf("%v\n", factory)
			log.Printf("Not in map %s\n", name)
			return nil, errors.New("function not in table " + name)
		}

		itf := ff()
		v = reflect.ValueOf(itf)

		// Add the object as second node of !type. Next time w'll pick this object.
		n.Add(v)

		// If !init is defined, the Init(Graph) function is called on the instantiated type.
		nn := g.Node("!init")

		if nn != nil {
			vf := v.MethodByName("Init")
			if vf.IsValid() {
				v.MethodByName("Init").Call([]reflect.Value{reflect.ValueOf(nn)})
			}
		}
	} else {
		v = n.GetAt(1).This.(reflect.Value)
	}

	// exec: as per path
	//
	// p[i] is a function or field name
	// rest are arguments.

	// obj = p.GetAt(i)

	fn := p.GetAt(ix)

	if fn == nil {
		s := "No method " + fn.Text()
		return s, errors.New(s)
	}

	fname := fn.ThisString()

	// Check if it is a method
	me := v.MethodByName(fname)

	if !me.IsValid() {

		// Try field
		if v.Kind() == reflect.Struct {
			v = v.FieldByName(fname)
			if v.IsValid() {
				return v.Interface(), nil
			}
		}

		s := "No method " + fname
		return s, errors.New(s)
	}

	// Build arguments in the form []reflect.Value

	var vargs []reflect.Value

	// Prepare a pre-evaluated parameter array
	var args []interface{}
	for _, arg := range p.Out[ix+1].Out {
		args = append(args, context.EvalExpression(arg))

	}

	for i, arg := range args {
		if arg == nil {
			// No untyped nil support :-(
			vargs = append(vargs, reflect.Zero(me.Type().In(i)))
		} else {
			vargs = append(vargs, reflect.ValueOf(arg))
		}
	}

	if me.Type().NumIn() != len(args) {
		s := "Invalid number of arguments in method " + fname
		return "", errors.New(s)
	}

	log.Println("function", fname)
	for i := 0; i < me.Type().NumIn(); i++ {
		log.Println("-", me.Type().In(i).String())
		log.Println("+", reflect.TypeOf(args[i]))
	}
	for i, arg := range args {
		v := reflect.TypeOf(arg)
		if v == nil || me.Type().In(i).String() != v.String() {
			s := "Invalid argument for method " + fname
			return "", errors.New(s)
		}
	}

	// TODO: return 0..2 values
	vv := me.Call(vargs)
	log.Println("len vv", len(vv))
	if len(vv) > 0 {
		return vv[0].Interface(), nil
	}
	return nil, nil
}

func init() {
	factory = make(map[string]func() interface{})
	factory["nil"] = nilGraphI

	functions = make(map[string]func(*Graph, []interface{}) []byte)
	functions["T"] = templateProcess
}

// Example functions and objects

func templateProcess(context *Graph, args []interface{}) []byte {
	if args == nil {
		return nil
	}
	s := _text(args[0])
	if len(s) == 0 {
		return nil
	}
	t := NewTemplate(s)
	return t.Process(context)
}

func nilGraphI() interface{} {
	return New()
}
