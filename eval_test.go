package ogdl

import (
	"fmt"
	"reflect"
	"testing"
)

func TestEval(t *testing.T) {

	cases := []struct {
		in   string
		expr string
		want string
	}{
		{"", "1-2+3", "2"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s in %s", tc.in, tc.want), func(t *testing.T) {
			g := FromString(tc.in)
			e := NewExpression(tc.expr)
			r, _ := g.Eval(e)
			got := _text(r)
			if got != tc.want {
				t.Errorf("got %s; want %s", got, tc.want)
			}
		})
	}
}

func TestEvalTypes(t *testing.T) {

	cases := []struct {
		in   string
		expr string
		want string
		typ  string
	}{
		// $a.(b), where a is a graph and b is a variable containing a path
		{"a c 1\nb c", "a.(b)", "1", "int64"},
		{"a '4'\nb 4", "a", "4", "string"},
		{"a '4'\nb 4", "b", "4", "int64"},
		{"a '4'\nb 4", "a+3", "43", "string"},
		// {"a '4'\nb 4", "a-3", "1", "int64"},  This tests fails!
		{"a '4'\nb 4", "b+3", "7", "int64"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s in %s", tc.in, tc.want), func(t *testing.T) {
			g := FromStringTypes(tc.in)
			e := NewExpression(tc.expr)
			r, _ := g.Eval(e)
			got := _text(r)
			gotType := _typeOf(r)
			if got != tc.want || gotType != tc.typ {
				t.Errorf("got %s; want %s", got, tc.want)
			}
		})
	}
}

func TestEvalPath(t *testing.T) {

	cases := []struct {
		in   string
		path string
		want string
	}{
		{"a\n b\n  c\n  d", "a", "b\n  c\n  d"},
		{"a\n b\n  c\n  d", "a[0]", "b\n  c\n  d"},
		{"a\n b 1\n b 2", "a.b[0]", "1"},
		{"a\n b 1\n b 2", "a.b", "1"},
		{"a\n b 1\n b 2", "a[0]", "b\n  1"},
		{"a\n b 1\n b 2", "a.b{1}", "2"},
		{"a\n b 1\n b 2", "a.b{}", "1\n2"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s in %s", tc.in, tc.want), func(t *testing.T) {
			g := FromString(tc.in)
			p := NewPath(tc.path)
			r, _ := g.evalPath(p, false)
			got := _text(r)
			if got != tc.want {
				t.Errorf("got %s; want %s", got, tc.want)
			}
		})
	}
}

func TestEvalExpression(t *testing.T) {

	// The context
	g := FromString("a b")

	// 'a' is a string constant
	p := NewExpression("a=='b'")
	r, _ := g.Eval(p)

	if _typeOf(r) != "bool" || r != true {
		t.Error("a=='b'")
	}

	p = NewExpression("'a'!='b'")
	r, _ = g.Eval(p)
	if _typeOf(r) != "bool" || r != true {
		t.Error("'a'!='b'")
	}

	p = NewExpression("2>1")
	r, _ = g.Eval(p)
	if _typeOf(r) != "bool" || r != true {
		t.Error("'a'!='b'")
	}
	p = NewExpression("2>=2")
	r, _ = g.Eval(p)
	if _typeOf(r) != "bool" || r != true {
		t.Error("'a'!='b'")
	}
	p = NewExpression("1<2")
	r, _ = g.Eval(p)
	if _typeOf(r) != "bool" || r != true {
		t.Error("'a'!='b'")
	}
	p = NewExpression("1<=2")
	r, _ = g.Eval(p)
	if _typeOf(r) != "bool" || r != true {
		t.Error("'a'!='b'")
	}
	p = NewExpression("1<0")
	r, _ = g.Eval(p)
	if _typeOf(r) != "bool" || r != false {
		t.Error("'a'!='b'")
	}

	// logic
	e := "'false' || 'true'"
	p = NewExpression(e)
	r, _ = g.Eval(p)
	if _typeOf(r) != "bool" || r != true {
		t.Error(e)
	}
	e = "'true' && 'true'"
	p = NewExpression(e)
	r, _ = g.Eval(p)
	if _typeOf(r) != "bool" || r != true {
		t.Error(e)
	}

	// Assign
	g = New(nil)
	e = "a=1"
	p = NewExpression(e)
	g.Eval(p)

	if i, err := g.GetInt64("a"); i != 1 || err != nil {
		t.Error(e, _typeOf(i), _text(i), err.Error())
	}

	e = "a+=12"
	p = NewExpression(e)
	g.Eval(p)

	if i, _ := g.GetInt64("a"); i != 13 {
		t.Error(e, _typeOf(g), _text(g))
	}

	e = "a-=1"
	p = NewExpression(e)
	g.Eval(p)

	if i, _ := g.GetInt64("a"); i != 12 {
		t.Error(e, _typeOf(g), _text(g))
	}

	e = "a*=2"
	p = NewExpression(e)
	g.Eval(p)

	if i, _ := g.GetInt64("a"); i != 24 {
		t.Error(e, _typeOf(g), _text(g))
	}

	e = "a/=4"
	p = NewExpression(e)
	g.Eval(p)

	if i, _ := g.GetInt64("a"); i != 6 {
		t.Error(e, _typeOf(g), _text(g))
	}

	e = "a%=4"
	p = NewExpression(e)
	g.Eval(p)

	if i, _ := g.GetInt64("a"); i != 2 {
		t.Error(e, _typeOf(g), _text(g))
	}

	// b non existent
	e = "b+=1"
	p = NewExpression(e)
	g.Eval(p)

	if i, _ := g.GetInt64("b"); i != 1 {
		t.Error(e, _typeOf(g), _text(g))
	}

	// c non existent
	e = "c-=1"
	p = NewExpression(e)
	g.Eval(p)

	if i, _ := g.GetInt64("c"); i != -1 {
		t.Error(e, _typeOf(g), _text(g))
	}

	// d non existent
	e = "d*=1"
	p = NewExpression(e)
	g.Eval(p)

	if i, _ := g.GetInt64("d"); i != 0 {
		t.Error(e, _typeOf(g), _text(g))
	}

	// e non existent
	e = "e/=1"
	p = NewExpression(e)
	g.Eval(p)

	if _, err := g.GetInt64("e"); err == nil {
		t.Error(e, _typeOf(g), _text(g))
	}

	e = "1+2"
	p = NewExpression(e)
	r, _ = g.Eval(p)

	if r != int64(3) {
		t.Error(e, _typeOf(r), _text(r))
	}

	e = "1+2.0"
	p = NewExpression(e)
	r, _ = g.Eval(p)

	if r != float64(3) {
		t.Error(e, _typeOf(r), _text(r))
	}

	e = "1.0+2.0"
	p = NewExpression(e)
	r, _ = g.Eval(p)

	if r != float64(3) {
		t.Error(e, _typeOf(r), _text(r))
	}

	e = "2-4"
	p = NewExpression(e)
	r, _ = g.Eval(p)

	if r != int64(-2) {
		t.Error(e, _typeOf(r), _text(r))
	}

	e = "10*3"
	p = NewExpression(e)
	r, _ = g.Eval(p)

	if r != int64(30) {
		t.Error(e, _typeOf(r), _text(r))
	}

	e = "10.0*3"
	p = NewExpression(e)
	r, _ = g.Eval(p)

	if r != float64(30) {
		t.Error(e, _typeOf(r), _text(r))
	}

	e = "10*3.0"
	p = NewExpression(e)
	r, _ = g.Eval(p)

	if r != float64(30) {
		t.Error(e, _typeOf(r), _text(r))
	}

	e = "10/3"
	p = NewExpression(e)
	r, _ = g.Eval(p)

	if r != int64(3) {
		t.Error(e, _typeOf(r), _text(r))
	}

	e = "10%3"
	p = NewExpression(e)
	r, _ = g.Eval(p)

	if r != int64(1) {
		t.Error(e, _typeOf(r), _text(r))
	}
}

func TestEvalBool(t *testing.T) {

	g := New(nil)

	p := NewExpression("1=='1'")
	r := g.evalBool(p)
	if r != true {
		t.Error("1=='1'")
	}

	p = NewExpression("true")
	r = g.evalBool(p)
	if r != false {
		t.Error("true as a path")
	}

	p = NewExpression("'true'")
	r = g.evalBool(p)
	if r != true {
		t.Error("'true' keyword")
	}
}

func TestEvalScalar(t *testing.T) {

	g := New(nil)
	p := New("1")

	// constants
	i, _ := g.Eval(p)
	s := reflect.TypeOf(i).String()
	if i != int64(1) || s != "int64" {
		t.Error("Eval constant")
	}

	p = New("1.1")
	i, _ = g.Eval(p)
	s = reflect.TypeOf(i).String()
	if i != float64(1.1) || s != "float64" {
		t.Error("Eval constant int")
	}

	p = New(1.1)
	i, _ = g.Eval(p)
	s = reflect.TypeOf(i).String()
	if i != float64(1.1) || s != "float64" {
		t.Error("Eval constant float")
	}

	p = New('c')
	i, _ = g.Eval(p)
	s = reflect.TypeOf(i).String()
	if i != int64('c') || s != "int64" {
		t.Error("Eval constant char")
	}

	p = New("true")
	i, _ = g.Eval(p)
	s = reflect.TypeOf(i).String()
	if i != true || s != "bool" {
		t.Error("Eval constant bool")
	}

	p = New(true)
	i, _ = g.Eval(p)
	s = reflect.TypeOf(i).String()
	if i != true || s != "bool" {
		t.Error("Eval constant bool 2")
	}
}

func TestCompare(t *testing.T) {

	b := compare(1, 1, '=')
	if !b {
		t.Error("compare =")
	}
	b = compare(1, 1.0, '=')
	if !b {
		t.Error("compare =")
	}
	b = compare(1.0, 1.0, '=')
	if !b {
		t.Error("compare =")
	}

	b = compare(1, 2, '<')
	if !b {
		t.Error("compare 3")
	}
	b = compare(1.0, 2, '<')
	if !b {
		t.Error("compare 4")
	}

	b = compare("a", "a", '=')
	if !b {
		t.Error("compare =")
	}
	b = compare("a", "a", '!')
	if b {
		t.Error("compare =")
	}

	b = compare(2, 1, '>')
	if !b {
		t.Error("compare >")
	}
	b = compare(2.0, 1.0, '>')
	if !b {
		t.Error("compare >")
	}

	// <=
	b = compare(1, 2, '-')
	if !b {
		t.Error("compare <=")
	}
	b = compare(2, 2, '-')
	if !b {
		t.Error("compare <=")
	}
	b = compare(2.0, 2.0, '-')
	if !b {
		t.Error("compare <=")
	}

	// >=
	b = compare(2, 1, '+')
	if !b {
		t.Error("compare >=")
	}
	b = compare(2, 2, '+')
	if !b {
		t.Error("compare >=")
	}
	b = compare(2.0, 2.0, '+')
	if !b {
		t.Error("compare >=")
	}

	// Cannot compare non numbers
	b = compare("a", 1, '=')
	if b {
		t.Error("compare a string")
	}
	b = compare(1, "a", '=')
	if b {
		t.Error("compare a string")
	}

	// !
	b = compare(1, 2, '!')
	if !b {
		t.Error("compare !=")
	}
	b = compare(1, 1, '!')
	if b {
		t.Error("compare !=")
	}
	b = compare(1.0, 2.0, '!')
	if !b {
		t.Error("compare !=")
	}
	b = compare(1.0, 1.0, '!')
	if b {
		t.Error("compare !=")
	}
}

// eval.go

func TestEvalCalcMod(t *testing.T) {

	i := calc(11.0, 2.0, '%')

	s := reflect.TypeOf(i).String()

	// % operates on ints!
	if s != "int" || i != 1 {
		t.Error("Calc %")
	}
}

func TestEvalCalcStr(t *testing.T) {

	i := calc("11.0-", 2.0, '+')

	s := reflect.TypeOf(i).String()

	if s != "string" || i != "11.0-2" {
		t.Error("string with numbers")
	}
}
