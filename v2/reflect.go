package ogdl

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
)

// This file implements the host interface of the expression RFC, section 8:
// reaching methods, fields and map entries of Go values held in the tree, and
// calling functions and methods.
//
// Everything here is reflection over values the host program itself placed in
// the tree. Nothing reaches a value the host did not provide.

var errorType = reflect.TypeOf((*error)(nil)).Elem()
var nodeType = reflect.TypeOf((*Node)(nil))

// isNilValue reports whether v is nil, or a typed nil pointer, interface, map,
// slice or function. The plain `v == nil` test misses a typed nil, which is how
// a nil *T stored in an interface silently passes for a value.
func isNilValue(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return rv.IsNil()
	}
	return false
}

// memberOf resolves a member of a host value by name: a method, then an
// exported struct field, then a map entry (Expr RFC 8.2). Child nodes are tried
// before this, by the caller, so the tree always shadows the host value.
func memberOf(v interface{}, name string) (interface{}, bool) {
	if v == nil || name == "" {
		return nil, false
	}
	if _, isNode := v.(*Node); isNode {
		// Nodes are navigated as trees, not by reflection.
		return nil, false
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, false
	}

	// A method may be defined on the value or on its pointer.
	if m := rv.MethodByName(name); m.IsValid() {
		return m.Interface(), true
	}
	if rv.Kind() != reflect.Ptr && rv.CanAddr() {
		if m := rv.Addr().MethodByName(name); m.IsValid() {
			return m.Interface(), true
		}
	}

	// Dereference for field and map access.
	ev := rv
	for ev.Kind() == reflect.Ptr || ev.Kind() == reflect.Interface {
		if ev.IsNil() {
			return nil, false
		}
		ev = ev.Elem()
	}

	switch ev.Kind() {
	case reflect.Struct:
		f := ev.FieldByName(name)
		// CanInterface is false for unexported fields, which MUST stay
		// unreachable (Expr RFC 8.2).
		if f.IsValid() && f.CanInterface() {
			return f.Interface(), true
		}
	case reflect.Map:
		kt := ev.Type().Key()
		if kt.Kind() == reflect.String {
			key := reflect.ValueOf(name)
			if kt != key.Type() {
				key = key.Convert(kt)
			}
			if e := ev.MapIndex(key); e.IsValid() {
				return e.Interface(), true
			}
		}
	}

	return nil, false
}

// indexOf indexes a host slice, array or string.
func indexOf(v interface{}, k int) (interface{}, error) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.String:
		if k >= rv.Len() {
			return nil, nil
		}
		if rv.Kind() == reflect.String {
			return string([]rune(rv.String())[k]), nil
		}
		return rv.Index(k).Interface(), nil
	}
	return nil, ErrIncompatibleType
}

// callAny calls a function or method value with the given arguments
// (Expr RFC 8.3).
func callAny(fn interface{}, args []interface{}) (result interface{}, err error) {
	return callWith(fn, args, false)
}

// callWith is callAny, or with v1 set, a call as ogdl v1 made it: missing
// arguments are zero values, extra ones are dropped, and an argument that
// cannot be converted is passed as the zero value of its parameter.
func callWith(fn interface{}, args []interface{}, v1 bool) (result interface{}, err error) {
	if fn == nil {
		return nil, ErrNotCallable
	}
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return nil, ErrNotCallable
	}
	if v.IsNil() {
		return nil, ErrNotCallable
	}

	t := v.Type()

	if v1 {
		n := t.NumIn()
		if t.IsVariadic() {
			n--
		}
		for len(args) < n {
			args = append(args, nil)
		}
		if !t.IsVariadic() && len(args) > n {
			args = args[:n]
		}
	}

	// Arity is checked before the call: Call panics on a mismatch, and a panic
	// here would be reported as a host fault rather than as the caller's error.
	if t.IsVariadic() {
		if len(args) < t.NumIn()-1 {
			return nil, fmt.Errorf("%w: got %d, want at least %d", ErrArgCount, len(args), t.NumIn()-1)
		}
	} else if len(args) != t.NumIn() {
		return nil, fmt.Errorf("%w: got %d, want %d", ErrArgCount, len(args), t.NumIn())
	}

	vargs := make([]reflect.Value, 0, len(args))
	for i, a := range args {
		var want reflect.Type
		switch {
		case t.IsVariadic() && i >= t.NumIn()-1:
			want = t.In(t.NumIn() - 1).Elem()
		default:
			want = t.In(i)
		}
		var cv reflect.Value
		if v1 {
			cv = convertArgV1(a, want)
		} else if cv, err = convertArg(a, want); err != nil {
			return nil, fmt.Errorf("argument %d: %w", i+1, err)
		}
		vargs = append(vargs, cv)
	}

	// A panic in host code is the host's fault, but it must not take down the
	// caller of the evaluator (Expr RFC 8.3).
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("panic in call: %v", r)
		}
	}()

	out := v.Call(vargs)

	switch len(out) {
	case 0:
		return nil, nil
	case 1:
		return out[0].Interface(), nil
	case 2:
		if out[1].Type().Implements(errorType) {
			if !out[1].IsNil() {
				return nil, out[1].Interface().(error)
			}
			return out[0].Interface(), nil
		}
	}
	return nil, ErrBadResults
}

// integerArg returns a number argument as an integer. ok reports whether the
// argument is a number at all; err, whether it is one an integer cannot
// represent, a fraction or out of range, which must not be truncated into a
// different value (Expr RFC 8.3).
func integerArg(a interface{}) (i int64, ok bool, err error) {
	i, f, isInt, ok := toNumber(a)
	if !ok || isInt {
		return i, ok, nil
	}
	if f != math.Trunc(f) || f < math.MinInt64 || f >= math.MaxInt64 {
		return 0, true, fmt.Errorf("%w: %g is not an integer", ErrIncompatibleType, f)
	}
	return int64(f), true, nil
}

// convertArg converts an argument to a parameter type (Expr RFC 8.3). An
// argument that cannot be converted is an error: substituting a zero value
// would call the host with data it never received.
func convertArg(a interface{}, want reflect.Type) (reflect.Value, error) {
	if a == nil {
		return reflect.Zero(want), nil
	}

	// A node is passed through when the parameter wants one, and simplified
	// otherwise.
	if n, ok := a.(*Node); ok {
		if want == nodeType {
			return reflect.ValueOf(n), nil
		}
		a = scalarOf(n)
		if a == nil {
			return reflect.Zero(want), nil
		}
	}

	av := reflect.ValueOf(a)
	at := av.Type()

	if at == want {
		return av, nil
	}
	if at.AssignableTo(want) {
		return av, nil
	}

	switch want.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, ok, err := integerArg(a); ok {
			if err == nil && reflect.Zero(want).OverflowInt(i) {
				err = fmt.Errorf("%w: %d overflows %s", ErrIncompatibleType, i, want)
			}
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(i).Convert(want), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if i, ok, err := integerArg(a); ok {
			if err == nil && (i < 0 || reflect.Zero(want).OverflowUint(uint64(i))) {
				err = fmt.Errorf("%w: %d overflows %s", ErrIncompatibleType, i, want)
			}
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(uint64(i)).Convert(want), nil
		}
	case reflect.Float32, reflect.Float64:
		if f, ok := toFloat64(a); ok {
			if reflect.Zero(want).OverflowFloat(f) {
				return reflect.Value{}, fmt.Errorf("%w: %g overflows %s", ErrIncompatibleType, f, want)
			}
			return reflect.ValueOf(f).Convert(want), nil
		}
	case reflect.Bool:
		if s, ok := a.(string); ok {
			b, err := strconv.ParseBool(s)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("%w: %q is not a boolean", ErrIncompatibleType, s)
			}
			return reflect.ValueOf(b), nil
		}
		return reflect.ValueOf(toBool(a)), nil
	case reflect.String:
		return reflect.ValueOf(toString(a)).Convert(want), nil
	case reflect.Interface:
		if want.NumMethod() == 0 {
			return reflect.ValueOf(a), nil
		}
		if at.Implements(want) {
			return av, nil
		}
	}

	if at.ConvertibleTo(want) {
		return av.Convert(want), nil
	}
	return reflect.Value{}, fmt.Errorf("%w: cannot use %s as %s", ErrIncompatibleType, at, want)
}

// convertArgV1 converts an argument as ogdl v1 did: a node is passed when
// the parameter wants one, and otherwise stands for the value of its first
// child (none for a leaf); what cannot be converted is the zero value.
func convertArgV1(a interface{}, want reflect.Type) reflect.Value {
	if n, ok := a.(*Node); ok && n != nil {
		if want == nodeType {
			return reflect.ValueOf(n)
		}
		if len(n.Out) == 0 {
			return reflect.Zero(want)
		}
		a = n.Out[0].Data
	}
	if a == nil {
		return reflect.Zero(want)
	}
	cv, err := convertArg(a, want)
	if err != nil {
		return reflect.Zero(want)
	}
	return cv
}
