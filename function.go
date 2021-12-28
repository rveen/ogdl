// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
)

// Function enables calling Go functions from templates.
//
// g is the Function's context. g.This contains the presumed class name.
// The _type subnode of g, if present, contains the function type (a Go
// interface name or 'rfunction'
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
// TODO: Catch panic() att Call(). Return named variables so that defer/recover
// returns something useful

func (g *Graph) function(path *Graph, typ interface{}) (interface{}, error) {

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Ogdl.function %s\n%s", err, path.Text())
			return
		}
	}()

	// log.Printf("\n%s\n", path.Show())

	v := reflect.ValueOf(typ)

	// Remote functions have this signature
	var f func(*Graph) (*Graph, error)
	rfType := reflect.ValueOf(f).Type()

	// Build arguments in the form []reflect.Value
	var vargs []reflect.Value

	switch v.Kind() {

	case reflect.Func:

		//log.Println("function.Func", path.Out[1].ThisString(), path.Out[1].Len())
		// log.Println("Func type", v.Type())
		//log.Println(runtime.FuncForPC(v.Pointer()).Name())
		//log.Println(reflect.TypeOf(typ).String())

		// Pre-evaluate
		var args []interface{}

		if v.Type() == rfType {
			// Remote function
			n := New("_")
			nn := n.Add(path.Out[1].This)
			if len(path.Out) > 2 {
				for _, arg := range path.Out[2].Out {
					// log.Printf("arg:\n%s\n", arg.Show())
					itf, _ := g.evalExpression(arg)
					nn.Add(itf)
				}
			}

			args = append(args, n)
		} else {
			// Local function
			for _, arg := range path.Out[1].Out {
				itf, _ := g.evalExpression(arg)
				args = append(args, itf)
				// log.Printf("%v\n", args[len(args)-1])
			}
		}

		for i, arg := range args {
			if arg == nil {
				// No untyped nil support :-(
				vargs = append(vargs, reflect.Zero(v.Type().In(i)))
			} else {
				vargs = append(vargs, reflect.ValueOf(arg))
			}
		}

		/* DEBUG CODE
		for i := 0; i < v.Type().NumIn(); i++ {
			log.Println("> ", v.Type().In(i).String())
		}
		for i := 0; i < len(vargs); i++ {
			log.Println("< ", vargs[i].Type().String())
		} /**/

		if v.Type().NumIn() != len(args) {
			// TODO Check that we print the name of the function
			return nil, fmt.Errorf("Invalid number of arguments in function %s (is %d, soll %d)\n%s", runtime.FuncForPC(v.Pointer()).Name(), len(args), v.Type().NumIn(), path.Show())
		}

		// TODO: return 0..2 values
		vv := v.Call(vargs)
		if len(vv) > 0 {
			return vv[0].Interface(), nil
		}
		return nil, nil

	case reflect.Ptr:

		// log.Println("function.Ptr")

		fn := path.GetAt(1)
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

			return nil, errors.New("No method or field: " + fname)
		}

		// Pre-evaluate
		var args []interface{}
		if len(path.Out) > 2 {
			for _, arg := range path.Out[2].Out {
				itf, _ := g.evalExpression(arg)
				args = append(args, itf)
			}
		}

		for i := 0; i < me.Type().NumIn(); i++ {
			mtype := me.Type()
			if i >= len(args) || args[i] == nil {
				// No untyped nil support :-(
				vargs = append(vargs, reflect.Zero(mtype.In(i)))
				continue
			}

			// Type adapter. A bit slow (cache could help)
			dtype := mtype.In(i).String()
			stype := reflect.TypeOf(args[i]).String()

			if dtype == stype {
				vargs = append(vargs, reflect.ValueOf(args[i]))
				continue
			}

			if stype == "string" && dtype == "float64" {
				v, _ := strconv.ParseFloat(args[i].(string), 64)
				vargs = append(vargs, reflect.ValueOf(v))
			} else if stype == "string" && dtype == "bool" {
				v, _ := strconv.ParseBool(args[i].(string))
				vargs = append(vargs, reflect.ValueOf(v))
			} else if stype == "string" && dtype == "int64" {
				v, _ := strconv.ParseInt(args[i].(string), 10, 64)
				vargs = append(vargs, reflect.ValueOf(v))
			} else if stype == "int64" && dtype == "float64" {
				vargs = append(vargs, reflect.ValueOf(float64(args[i].(int64))))
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
