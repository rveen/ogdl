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

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// function enables calling Go functions from templates. It evaluates 'path'
// in the context of g, that is, the context in which the function arguments are
// evaluated.
func (g *Graph) function(path *Graph, typ interface{}) (result interface{}, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in function call: %v", r)
		}
	}()

	v := reflect.ValueOf(typ)

	// Remote functions have this signature
	var f func(*Graph) (*Graph, error)
	rfType := reflect.ValueOf(f).Type()

	// Build arguments in the form []reflect.Value
	var vargs []reflect.Value

	switch v.Kind() {

	case reflect.Func:

		// Pre-evaluate
		var args []interface{}

		if v.Type() == rfType {
			// Remote function
			n := New("_")
			nn := n.Add(path.Out[1].This)
			if len(path.Out) > 2 {
				for _, arg := range path.Out[2].Out {
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
			}
		}

		// Check that the argument types match, otherwise v.Call() will panic
		if v.Type().NumIn() != len(args) {
			return nil, fmt.Errorf("Invalid number of arguments in function %s (is %d, soll %d)\n%s", runtime.FuncForPC(v.Pointer()).Name(), len(args), v.Type().NumIn(), path.Show())
		}

		for i, arg := range args {
			if arg == nil {
				// No untyped nil support :-(
				vargs = append(vargs, reflect.Zero(v.Type().In(i)))
			} else {
				vargs = append(vargs, convert(arg, v.Type().In(i)))
			}
		}

		for i := 0; i < v.Type().NumIn(); i++ {
			expected := v.Type().In(i)
			actual := vargs[i].Type()
			if actual == expected {
				// exact match
			} else if actual.AssignableTo(expected) {
				// interface satisfaction or same underlying type — ok as-is
			} else if actual.ConvertibleTo(expected) {
				vargs[i] = vargs[i].Convert(expected)
			} else {
				return nil, fmt.Errorf("argument %d: cannot use %s as %s", i, actual, expected)
			}
		}

		vv := v.Call(vargs)
		if len(vv) == 0 {
			return nil, nil
		}
		if len(vv) == 2 && vv[1].Type().Implements(errorType) && !vv[1].IsNil() {
			return nil, vv[1].Interface().(error)
		}
		return vv[0].Interface(), nil

	case reflect.Ptr:

		fn := path.GetAt(1)
		if fn == nil {
			return nil, errors.New("No method")
		}
		fname := fn.ThisString()

		// Check if it is a method
		me := v.MethodByName(fname)

		if !me.IsValid() {
			// Try field on dereferenced pointer
			elem := v.Elem()
			if elem.Kind() == reflect.Struct {
				f := elem.FieldByName(fname)
				if f.IsValid() {
					return f.Interface(), nil
				}
			}

			return nil, errors.New("No method or field: " + fname)
		}

		// Pre-evaluate
		var args []interface{}
		if len(path.Out) > 2 {
			for _, arg := range path.Out[2].Out {
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

			vargs = append(vargs, convert(args[i], mtype.In(i)))
		}

		for i := 0; i < me.Type().NumIn(); i++ {
			expected := me.Type().In(i)
			actual := vargs[i].Type()
			if actual == expected {
				// exact match
			} else if actual.AssignableTo(expected) {
				// interface satisfaction or same underlying type — ok as-is
			} else if actual.ConvertibleTo(expected) {
				vargs[i] = vargs[i].Convert(expected)
			} else {
				return nil, fmt.Errorf("argument %d: cannot use %s as %s", i, actual, expected)
			}
		}

		vv := me.Call(vargs)
		if len(vv) == 0 {
			return nil, nil
		}
		if len(vv) == 2 && vv[1].Type().Implements(errorType) && !vv[1].IsNil() {
			return nil, vv[1].Interface().(error)
		}
		return vv[0].Interface(), nil

	default:
		return nil, nil
	}

}

// convert converts arg to targetType, if possible.
func convert(arg interface{}, targetType reflect.Type) reflect.Value {

	dtype := targetType.String()
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

	return reflect.Zero(targetType)
}
