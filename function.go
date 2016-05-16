// Copyright 2012-2015, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"errors"
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

// Function enables calling Go functions from templates.
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
// Example 1
//
// !p
//   T
//   !g
//     'some text'
//
// Example 2
// !p
//   math
//   Sin
//   !g
//    !e
//      1.0
//
// Functions calls are limited to whole paths.
//
func (g *Graph) function(path *Graph, ix int, typ interface{}) (interface{}, error) {

	v := reflect.ValueOf(typ)

	// Build arguments in the form []reflect.Value
	var vargs []reflect.Value

	switch v.Kind() {

	case reflect.Func:

		// Pre-evaluate
		var args []interface{}
		for _, arg := range path.Out[ix].Out {
			args = append(args, g.evalExpression(arg))

		}

		for i, arg := range args {
			if arg == nil {
				// No untyped nil support :-(
				vargs = append(vargs, reflect.Zero(v.Type().In(i)))
			} else {
				vargs = append(vargs, reflect.ValueOf(arg))
			}
		}

		if v.Type().NumIn() != len(args) {
			s := "Invalid number of arguments in function"
			return nil, errors.New(s)
		}

		// TODO: return 0..2 values
		vv := v.Call(vargs)
		if len(vv) > 0 {
			return vv[0].Interface(), nil
		}
		return nil, nil

	case reflect.Ptr:

		fn := path.GetAt(ix)
		if fn == nil {
			return nil, errors.New("No method")
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

			return nil, errors.New("No method: " + fname)
		}

		// Pre-evaluate
		var args []interface{}
		for _, arg := range path.Out[ix+1].Out {
			args = append(args, g.evalExpression(arg))

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
			return nil, errors.New("Invalid number of arguments in method " + fname)
		}

		for i, arg := range args {
			v := reflect.TypeOf(arg)
			if v == nil || me.Type().In(i).String() != v.String() {
				return nil, errors.New("Invalid argument for method " + fname)
			}
		}

		// TODO: return 0..2 values
		vv := me.Call(vargs)
		if len(vv) > 0 {
			return vv[0].Interface(), nil
		}
		return nil, nil

	default:
		return nil, nil
	}

}
