// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

// GetSimilar returns a Graph with all subnodes
// found that match the regular expression given.
//
func (g *Graph) GetSimilar(re string) (*Graph, bool) {
	exp, error := regexp.Compile(re)

	if error != nil {
		return nil, false
	}

	r := NilGraph()

	for _, node := range g.Out {
		if exp.MatchString(node.String()) {
			r.Add(node)
		}
	}

	return r, true
}

// Return the node as an int64, if possible.
func (g *Graph) Int64() (int64, bool) {
	return _int64f(g.String())
}

// Return the node as a float64, if possible.
func (g *Graph) Float64() (float64, bool) {
	return _float64f(g.String())
}

// Return the node as a boolean, if possible.
func (g *Graph) Bool() (bool, bool) {
	return _boolf(g.String())
}

// Return the node as a reflect.Value.
func (g *Graph) Value() reflect.Value {
	return reflect.ValueOf(g.This)
}

// String returns a string representation of this node.
//
// This function is specially used when comparing path elements (which are
// strings) in Get.
func (g *Graph) String() string {
	return _string(g.This)
}

// String returns a the node as []byte, or nil if not possble.
func (g *Graph) Bytes() []byte {
	return _bytes(g.This)
}

// Number returns either a float64, int64 or nil
// In strings and byte arrays, valid digits are ASCII only
//
func (g *Graph) Number() interface{} {
	return number(g.This)
}

func number(itf interface{}) interface{} {

	if itf == nil {
		return nil
	}

	i, ok := _int64(itf)
	if ok {
		return i
	}

	f, ok := _float64(itf)
	if ok {
		return f
	}

	s := _string(itf)
	if len(s) == 0 {
		return nil
	}

	if IsInteger(s) {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil
		}
		return n
	}

	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return n
}

// GetString returns the result of applying a path to the given Graph.
func (g *Graph) GetString(path string) (string, bool) {

	i := g.Get(path)
	return _string(i), true
}

func (g *Graph) GetBytes(s string) ([]byte, bool) {

	i := g.Get(s)
	return _bytes(i), true
}

func (g *Graph) GetInt64(s string) (int64, bool) {
	return _int64f(g.Get(s))
}

func (g *Graph) GetFloat64(s string) (float64, bool) {
	return _float64f(g.Get(s))
}

func (g *Graph) GetBool(s string) (bool, bool) {
	return _boolf(g.Get(s))
}

func _float64f(v interface{}) (float64, bool) {

	f, ok := _float64(v)
	if ok {
		return f, true
	}

	i, ok := _int64(v)
	if ok {
		return float64(i), true
	}

	s := _string(v)
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f, true
	}

	return 0, false
}

// _float64 converts an interface{} to a float64 iff its native type is
// a floating point.
func _float64(i interface{}) (float64, bool) {

	switch v := i.(type) {

	case float32:
		return float64(v), true
	case float64:
		return v, true
	}

	return 0, false
}

// _int64f converts an interface{} to an int64 if its native type is
// a integer (whether 8, 16, 32 or 64 bits), or can be converted to one.
func _int64f(i interface{}) (int64, bool) {

	if i2, ok := i.(*Graph); ok {
		i = i2.This
	}

	switch v := i.(type) {
	case []byte:
		n, error := strconv.ParseInt(string(v), 10, 64)
		if error == nil {
			return n, true
		}
	case string:
		n, error := strconv.ParseInt(v, 10, 64)
		if error == nil {
			return n, true
		}
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	}

	return _int64(i)
}

// _int64 converts an interface{} to an int64 if its native type is
// a integer (whether 8, 16, 32 or 64 bits, rune or byte).
func _int64(i interface{}) (int64, bool) {

	switch v := i.(type) {

	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	}

	return 0, false
}

func _boolf(i interface{}) (bool, bool) {

	if i2, ok := i.(*Graph); ok {
		i = i2.This
	}

	switch v := i.(type) {
	case []byte:
		s := string(v)

		if "false" == s {
			return false, true
		}
		if "true" == s {
			return true, true
		}
		return false, false
	case string:
		if "false" == v {
			return false, true
		}
		if "true" == v {
			return true, true
		}
		return false, false
	case bool:
		return v, true
	}

	return false, false
}

func _string(i interface{}) string {
	return string(_bytes(i))
}

func _bytes(i interface{}) []byte {
	if i == nil {
		return nil
	}
	if v, ok := i.([]byte); ok {
		return v
	}
	if v, ok := i.(string); ok {
		return []byte(v)
	}
	if v, ok := i.(*Graph); ok {
		return []byte(v.String())
	}
	return []byte(fmt.Sprint(i))
}

// Scalar reduces the number of types following these rules:
//
// uint* -> int64
// int*  -> int64
// float* -> float64
// byte -> int64
// rune -> int64
// bool -> bool
// string, []byte: if it represents an int or float or bool,
//     convert to int64, float64 or bool
// Any other type is returned as is.
//
// TODO: Evaluate if type reduction is really necessary. For now it simplifies
// the eval functions.
func (g *Graph) Scalar() interface{} {
	return scalar(g.This)
}

func scalar(itf interface{}) interface{} {
	n := number(itf)
	if n != nil {
		return n
	}
	b, ok := _boolf(itf)
	if ok {
		return b
	}
	return itf
}

// XXX used in eval.go
func isNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	if !IsDigit(int(s[0])) {
		if len(s) < 2 || s[0] != '-' || !IsDigit(int(s[1])) {
			return false
		}
	}
	return true
} 

// IsInteger returns true for strings containing exclusively digits, with an
// optional minus sign at the beginning. Starting and trailing spaces are 
// allowed.
func IsInteger(s string) bool {

	l := len(s)

	if l == 0 {
		return false
	}

	i := 0
	
	for ; i < l; i++ {
	    if !IsSpaceChar(int(s[i])) {
	        break;
	    }
	}

	if s[i] == '-' {
		i++
	}

    n := 0
	for ; i < l; i++ {
		if !IsDigit(int(s[i])) {
			break
		}
		n++
	}
	
	if n==0 {
	    return false
	}
	
	for ; i < l; i++ {
	    if !IsSpaceChar(int(s[i])) {
	        return false
	    }
	}

	return true
}
