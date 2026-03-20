// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
)

// Find returns a Graph with all subnodes that match the regular
// expression given. It only walks through the subnodes of the current Graph.
// If the regex doesn't compile, an error will be returned. If the result set
// is empty, both return values are nil (no error is signaled).
func (g *Graph) Find(re string) (*Graph, error) {
	exp, err := regexp.Compile(re)

	if err != nil {
		return nil, err
	}

	r := New("_")

	for _, node := range g.Out {
		if exp.MatchString(node.ThisString()) {
			r.Add(node)
		}
	}

	return r, nil
}

// Int64 returns the node as an int64. If the node is not a number, it
// returns 0, or the default value if given.
func (g *Graph) Int64(def ...int64) int64 {
	n, ok := _int64f(g.Interface())
	if !ok {
		if len(def) == 0 {
			return 0
		}
		return def[0]
	}
	return n
}

// Float64 returns the node as a float64. If the node is not a number, it
// return NaN, or the default value if given.
func (g *Graph) Float64(def ...float64) float64 {
	n, ok := _float64f(g.String())
	if !ok {
		if len(def) == 0 {
			return math.NaN()
		}
		return def[0]
	}
	return n
}

// Bool returns the node as a boolean. If the node is not a
// boolean, it returns false, or the default value if given.
func (g *Graph) Bool(def ...bool) bool {
	n, ok := _boolf(g.String())
	if !ok {
		if len(def) == 0 {
			return false
		}
		return def[0]
	}
	return n
}

// ThisValue returns this node as a reflect.Value.
func (g *Graph) ThisValue() reflect.Value {
	return reflect.ValueOf(g.This)
}

// Value returns the node as a reflect.Value.
func (g *Graph) Value() reflect.Value {
	return reflect.ValueOf(g.Interface())
}

// String returns a string representation of this node, or an empty string.
// This function doesn't return an error, because it is mostly used in single
// variable return situations.
// String accepts one default value, which will be returned instead of an
// empty string.
func (g *Graph) String(def ...string) string {

	// If g is nil, return default or nothing
	if g == nil {
		if len(def) == 0 {
			return ""
		}
		return def[0]
	}

	return _string(g.Interface())
}

// StringCSV returns a comma separated value representation of all direct subnodes.
// StringCSV accepts one default value, which will be returned instead of an
// empty string.
func (g *Graph) StringCSV(def ...string) string {

	// If g is nil, return default or nothing
	if g == nil {
		if len(def) == 0 {
			return ""
		}
		return def[0]
	}

	s := ""

	for _, n := range g.Out {
		s += ", " + _string(n.This)
	}

	return s[2:]
}

// String returns an array of strings representing all direct subnodes.
func (g *Graph) Strings() []string {

	// If g is nil, return default or nothing
	if g == nil {
		return nil
	}
	/*
		if s, ok := g.This.([]string); ok {
			return s
		}
	*/
	var ss []string

	for _, n := range g.Out {
		ss = append(ss, _string(n.This))
	}

	return ss
}

// StringTxt returns a space separated value representation of all direct subnodes.
// StringTxt accepts one default value, which will be returned instead of an
// empty string.
func (g *Graph) StringTxt(def ...string) string {

	// If g is nil, return default or nothing
	if g == nil {
		if len(def) == 0 {
			return ""
		}
		return def[0]
	}

	if g.Len() == 0 {
		return ""
	}

	s := ""

	for _, n := range g.Out {
		s += " " + _string(n.This)
	}

	return s[1:]
}

// ThisString returns the current node content as a string
func (g *Graph) ThisString(def ...string) string {

	// If g is nil, return default or nothing
	if g == nil {
		if len(def) == 0 {
			return ""
		}
		return def[0]
	}

	return _string(g.This)
}

// Bytes returns the graph as []byte, or nil if not possble.
func (g *Graph) Bytes() []byte {
	return _bytes(g.Interface())
}

// ThisBytes returns the node as []byte, or nil if not possble.
func (g *Graph) ThisBytes() []byte {
	return _bytes(g.This)
}

// Number returns either a float64, int64 or nil
func (g *Graph) Number() interface{} {
	return number(g.Interface())
}

// ThisNumber returns either a float64, int64 or nil
func (g *Graph) ThisNumber() interface{} {
	return number(g.This)
}

// ThisInt64 returns a int64 or nil
func (g *Graph) ThisInt64() (int64, bool) {
	return _int64f(g.This)
}

// ThisFloat64 returns a float64
func (g *Graph) ThisFloat64() (float64, bool) {
	return _float64f(g.This)
}

// Scalar returns the current node content, reducing the number of types
// following these rules:
//
//	uint* -> int64
//	int*  -> int64
//	float* -> float64
//	byte -> int64
//	rune -> int64
//	bool -> bool
//	string, []byte: if it represents an int or float or bool,
//	  convert to int64, float64 or bool
//
// Any other type is returned as is.
func (g *Graph) Scalar() interface{} {

	itf := g.Interface()
	if itf == nil && g.Out != nil {
		itf = g.Out[0].This
	}

	// If it can be parsed as a number, return it.
	n := number(itf)
	if n != nil {
		return n
	}

	// If it can be parsed as a bool, return it.
	b, ok := _boolf(itf)
	if ok {
		return b
	}

	// Else return as is.
	return itf
}

// ThisScalar returns this node's content as an interface
func (g *Graph) ThisScalar() interface{} {

	itf := g.This
	if itf == nil && g.Out != nil {
		itf = g.Out[0].This
	}

	// If it ca be parsed as a number, return it.
	n := number(itf)
	if n != nil {
		return n
	}

	// If it can be parsed as a bool, return it.
	b, ok := _boolf(itf)
	if ok {
		return b
	}

	// Else return as is.
	return itf
}

// Interface returns the first child of this node as an interface
func (g *Graph) Interface() interface{} {
	if g != nil && g.Out != nil && len(g.Out) != 0 {
		return g.Out[0].This
	}
	return nil
}

// number tries hard to convert the parameter to an int64 or float64. If it
// cannot, then it returns nil.
func number(itf interface{}) interface{} {

	if itf == nil {
		return nil
	}

	f, ok := _float64(itf)
	if ok {
		return f
	}

	i, ok := _int64(itf)
	if ok {
		return i
	}

	s := _string(itf)
	if len(s) == 0 {
		return nil
	}

	if isInteger(s) {
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
// The result is returned as a string.
// If the error information is not used, then this method is equivalent
// to Get(path).String()
func (g *Graph) GetString(path string) (string, error) {
	// If path is "", return .String()
	if len(path) == 0 {
		return g.String(), nil
	}
	i := g.Get(path)
	if i == nil {
		return "", errors.New("not found")
	}
	if i.Len() == 0 {
		return "", errors.New("Get() design error: not subnodes")
	}
	return _string(i.Out[0].This), nil
}

// GetBytes returns the result of applying a path to the given Graph.
// The result is returned as a byte slice.
func (g *Graph) GetBytes(path string) ([]byte, error) {

	if len(path) == 0 {
		return g.Bytes(), nil
	}

	i := g.Get(path)
	if i == nil {
		return nil, errors.New("not found")
	}
	if i.Len() == 0 {
		return nil, errors.New("Get() design error: not subnodes")
	}
	return _bytes(i.Out[0].This), nil
}

// GetInt64 returns the result of applying a path to the given Graph.
// The result is returned as an int64. If the path result cannot be converted
// to an integer, then an error is returned.
func (g *Graph) GetInt64(path string) (int64, error) {

	if len(path) == 0 {
		return g.Int64(), nil
	}

	i := g.Get(path)
	if i == nil {
		return 0, errors.New("not found")
	}
	if i.Len() == 0 {
		return 0, errors.New("Get() design error: not subnodes")
	}
	j, ok := _int64f(i.Out[0].This)
	if !ok {
		return 0, errors.New("incompatible type")
	}
	return j, nil
}

// GetFloat64 returns the result of applying a path to the given Graph.
// The result is returned as a float64. If the path result cannot be converted
// to a float, then an error is returned.
func (g *Graph) GetFloat64(path string) (float64, error) {

	if len(path) == 0 {
		return g.Float64(), nil
	}

	i := g.Get(path)
	if i == nil {
		return 0, errors.New("not found")
	}
	if i.Len() == 0 {
		return 0, errors.New("Get() design error: not subnodes")
	}
	j, ok := _float64f(i.Out[0].This)
	if !ok {
		return 0, errors.New("not a number")
	}
	return j, nil
}

// GetBool returns the result of applying a path to the given Graph.
// The result is returned as a bool. If the path result cannot be converted
// to a boolean, then an error is returned.
func (g *Graph) GetBool(path string) (bool, error) {

	if len(path) == 0 {
		return g.Bool(), nil
	}

	i := g.Get(path)
	if i == nil {
		return false, errors.New("not found")
	}
	if i.Len() == 0 {
		return false, errors.New("Get() design error: not subnodes")
	}
	j, ok := _boolf(i.Out[0].This)
	if !ok {
		return false, errors.New("not a boolean")
	}
	return j, nil
}

// _float64 converts an interface{} to a float64 iff its native type is
// a float, integer or a string representing a number.
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

// _boolf converts an interface{} to a boolean if possible.
func _boolf(i interface{}) (bool, bool) {

	if i2, ok := i.(*Graph); ok {
		// i2 can still be nil (of type *Graph!)
		if i2 == nil {
			return false, false
		}
		i = i2.This
	}

	switch v := i.(type) {
	case []byte:
		s := string(v)

		if falseStr == s {
			return false, true
		}
		if trueStr == s {
			return true, true
		}
		return false, false
	case string:
		if falseStr == v {
			return false, true
		}
		if trueStr == v {
			return true, true
		}
		return false, false
	case bool:
		return v, true
	case int:
		if v == 0 {
			return false, true
		}
		return true, true
	case int64:
		if v == 0 {
			return false, true
		}
		return true, true
	}

	return false, false
}

func _string(i interface{}) string {
	if i == nil {
		return ""
	}
	if v, ok := i.([]byte); ok {
		return string(v)
	}
	if v, ok := i.(string); ok {
		return v
	}
	if v, ok := i.(*Graph); ok {
		return v.String()
	}
	return fmt.Sprint(i)
}

// simplify graphs in the form:
// *
//
//	something
//
// to a scalar (return something)
func _simplify(i interface{}) interface{} {

	if v, ok := i.(*Graph); ok {
		if v.Len() == 1 && v.Out[0].Len() == 0 {
			return v.Out[0].This
		}
	}
	return i
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
		return []byte(v.ThisString())
	}
	return []byte(fmt.Sprint(i))
}

func _text(i interface{}) string {
	if i == nil {
		return ""
	}
	if v, ok := i.([]byte); ok {
		return string(v)
	}
	if v, ok := i.(string); ok {
		return v
	}
	if v, ok := i.(*Graph); ok {
		return v.Text()
	}
	return fmt.Sprint(i)
}

// _show: this function is used in tests
func _show(i interface{}) string {
	if i == nil {
		return ""
	}
	if v, ok := i.([]byte); ok {
		return string(v)
	}
	if v, ok := i.(string); ok {
		return v
	}
	if v, ok := i.(*Graph); ok {
		return v.Show()
	}
	return fmt.Sprint(i)
}

// _typeOf: this function is used in tests
func _typeOf(i interface{}) string {
	if i == nil {
		return ""
	}
	return reflect.TypeOf(i).String()
}

// isNumber is only used in eval.go
func isNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	if !IsDigit(rune(s[0])) {
		if len(s) < 2 || s[0] != '-' || !IsDigit(rune(s[1])) {
			return false
		}
	}
	return true
}

// IsInteger returns true for strings containing exclusively digits, with an
// optional minus sign at the beginning. Starting and trailing spaces are
// allowed.
func isInteger(s string) bool {

	l := len(s)

	if l == 0 {
		return false
	}

	i := 0

	for ; i < l; i++ {
		if !IsSpaceChar(s[i]) {
			break
		}
	}

	if s[i] == '-' {
		i++
	}

	n := 0
	for ; i < l; i++ {
		if !IsDigit(rune(s[i])) {
			break
		}
		n++
	}

	if n == 0 {
		return false
	}

	for ; i < l; i++ {
		if !IsSpaceChar(s[i]) {
			return false
		}
	}

	return true
}
