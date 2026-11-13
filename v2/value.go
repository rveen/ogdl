package ogdl

import "strconv"

// This file implements the value model of the expression RFC, section 2: the
// kinds a value may take, operand simplification, and the coercions between
// them. Coercion happens at the operator, never in advance.

// scalarOf applies operand simplification (Expr RFC 2.2): a node with no
// children stands for its own content, and a node with exactly one child, that
// child having no children of its own, stands for that child's content. Any
// other value is returned unchanged.
func scalarOf(v interface{}) interface{} {
	n, ok := v.(*Node)
	if !ok || n == nil {
		return v
	}
	switch len(n.Out) {
	case 0:
		return n.Data
	case 1:
		if len(n.Out[0].Out) == 0 {
			return n.Out[0].Data
		}
	}
	return v
}

// parseNumber returns the int64 or float64 a string denotes, if it denotes one
// completely (Expr RFC 3.1).
func parseNumber(s string) (interface{}, bool) {
	if s == "" {
		return nil, false
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, true
	}
	// ParseFloat also accepts hexadecimal forms, "Inf" and "NaN"; only decimal
	// numbers are numbers (Schema RFC 4.1).
	if f, err := strconv.ParseFloat(s, 64); err == nil && isDecimal(s) {
		return f, true
	}
	return nil, false
}

// toNumber coerces to int64 or float64 (Expr RFC 2.3). isInt reports which.
func toNumber(v interface{}) (i int64, f float64, isInt bool, ok bool) {
	switch x := scalarOf(v).(type) {
	case nil:
		return 0, 0, false, false
	case int:
		return int64(x), 0, true, true
	case int8:
		return int64(x), 0, true, true
	case int16:
		return int64(x), 0, true, true
	case int32:
		return int64(x), 0, true, true
	case int64:
		return x, 0, true, true
	case uint:
		return int64(x), 0, true, true
	case uint8:
		return int64(x), 0, true, true
	case uint16:
		return int64(x), 0, true, true
	case uint32:
		return int64(x), 0, true, true
	case uint64:
		return int64(x), 0, true, true
	case float32:
		return 0, float64(x), false, true
	case float64:
		return 0, x, false, true
	case string:
		return numberFromString(x)
	case []byte:
		return numberFromString(string(x))
	}
	return 0, 0, false, false
}

func numberFromString(s string) (int64, float64, bool, bool) {
	n, ok := parseNumber(s)
	if !ok {
		return 0, 0, false, false
	}
	if i, isInt := n.(int64); isInt {
		return i, 0, true, true
	}
	return 0, n.(float64), false, true
}

// toInt64 coerces to an integer, truncating a float.
func toInt64(v interface{}) (int64, bool) {
	i, f, isInt, ok := toNumber(v)
	if !ok {
		return 0, false
	}
	if isInt {
		return i, true
	}
	return int64(f), true
}

// toFloat64 coerces to a float.
func toFloat64(v interface{}) (float64, bool) {
	i, f, isInt, ok := toNumber(v)
	if !ok {
		return 0, false
	}
	if isInt {
		return float64(i), true
	}
	return f, true
}

// toString returns the textual form of any value (Expr RFC 2.3). Every value
// has one, so this cannot fail.
func toString(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(x), 'g', -1, 32)
	case *Node:
		if x == nil {
			return ""
		}
		return nameString(x.Data)
	}
	return nameString(v)
}

// toBool coerces to a boolean. It is total: there is no error case
// (Expr RFC 2.3).
func toBool(v interface{}) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case *Node:
		if x == nil {
			return false
		}
		if len(x.Out) > 0 {
			return true
		}
		return toBool(x.Data)
	case string:
		return boolFromString(x)
	case []byte:
		return boolFromString(string(x))
	}
	if i, f, isInt, ok := toNumber(v); ok {
		if isInt {
			return i != 0
		}
		return f != 0
	}
	return !isNilValue(v)
}

func boolFromString(s string) bool {
	switch s {
	case "true":
		return true
	case "false", "":
		return false
	}
	return true
}

// Int64 returns the text below this node as an integer, or the default.
func (n *Node) Int64(def ...int64) int64 {
	if v, ok := toInt64(n.scalar()); ok {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return 0
}

// Float64 returns the text below this node as a float, or the default.
func (n *Node) Float64(def ...float64) float64 {
	if v, ok := toFloat64(n.scalar()); ok {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return 0
}

// Bool returns the text below this node as a boolean, or the default when the
// node holds nothing.
func (n *Node) Bool(def ...bool) bool {
	if n == nil || len(n.Out) == 0 {
		if len(def) > 0 {
			return def[0]
		}
		return false
	}
	return toBool(n.Out[0].Data)
}

// Strings returns the textual form of each child.
func (n *Node) Strings() []string {
	if n == nil {
		return nil
	}
	out := make([]string, 0, len(n.Out))
	for _, m := range n.Out {
		out = append(out, nameString(m.Data))
	}
	return out
}

// scalar returns the content this node stands for: its first child's, which is
// where a document like "port 8080" keeps the value.
func (n *Node) scalar() interface{} {
	if n == nil {
		return nil
	}
	if len(n.Out) > 0 {
		return n.Out[0].Data
	}
	return n.Data
}
