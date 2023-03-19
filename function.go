// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"errors"
	"fmt"

	"log"
	"reflect"
	"runtime"
	"strconv"
)

// function enables calling Go functions from templates. It evaluates 'path'
// in the context of g, that is, the context in which the function arguments are
// evaluated.
//
// TODO Needs good explanation and clean-up.
//
// TODO: Catch panic() att Call(). Return named variables so that defer/recover
// returns something useful
//
func (g *Graph) function(path *Graph, typ interface{}) (interface{}, error) {

	defer func() {
		if err := recover(); err != nil {
			log.Printf("Ogdl.function %s | %s", err, path.String())
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
					itf, _ := g.evalExpression(arg, true)
					nn.Add(itf)
				}
			}

			args = append(args, n)
		} else {
			// Local function
			for _, arg := range path.Out[1].Out {
				itf, _ := g.evalExpression(arg, true)
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

		//log.Println("function.Ptr")

		fn := path.GetAt(1)
		if fn == nil {
			return nil, errors.New("No method")
		}
		fname := fn.ThisString()

		// Check if it is a method
		me := v.MethodByName(fname)

		// log.Println(" - fname:", fname, me.IsValid(), me.Type().NumIn())

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
				// log.Println(" - arg", arg.Text())
				itf, _ := g.evalExpression(arg, false)
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
			vargs = append(vargs, convert(args[i], dtype))
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

// Convert arg to dtype, if possible.
func convert(arg interface{}, dtype string) reflect.Value {

	stype := reflect.TypeOf(arg).String()

	if dtype == stype {
		return reflect.ValueOf(arg)
	}

	switch stype {

	case "*ogdl.Graph":
		n, ok := arg.(*Graph)
		if ok {
			switch dtype {
			case "int64":
				v := n.Int64()
				return reflect.ValueOf(v)
			case "bool":
				v := n.Bool()
				return reflect.ValueOf(v)
			case "string":
				v := n.String()
				return reflect.ValueOf(v)
			case "[]string":
				v := n.Strings()
				return reflect.ValueOf(v)
			case "float64":
				v := n.Float64()
				return reflect.ValueOf(v)
			case "int":
				v := int(n.Int64())
				return reflect.ValueOf(v)
			}
		}

	case "string":
		switch dtype {
		case "float64":
			v, _ := strconv.ParseFloat(arg.(string), 64)
			return reflect.ValueOf(v)
		case "bool":
			v, _ := strconv.ParseBool(arg.(string))
			return reflect.ValueOf(v)
		case "int64":
			v, _ := strconv.ParseInt(arg.(string), 10, 64)
			return reflect.ValueOf(v)
		}

	case "int64":
		switch dtype {
		case "float64":
			return reflect.ValueOf(float64(arg.(int64)))
		case "int":
			return reflect.ValueOf(int(arg.(int64)))
		}

	case "float64":
		switch dtype {
		case "int64":
			return reflect.ValueOf(int64(arg.(float64)))
		case "int":
			return reflect.ValueOf(int(arg.(float64)))
		}
	}

	return reflect.Zero(nil)
}
