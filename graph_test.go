package ogdl

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"testing"
)

// Test resistance agains nil
func TestAdd1(t *testing.T) {

}

// json

func TestJson1(t *testing.T) {
	g := FromString("a\n 1\nb\n c")
	b := g.JSON()

	fmt.Println(string(b))
}

/*
object:

a
  b
    x 1
    y 2
  c
    z 3

*/
func TestJson2(t *testing.T) {
	g := FromString("a\n b\n  x 1\n  y2\n c z 3")
	b := g.JSON()

	fmt.Println(string(b))
}

// Other parser tests

func TestUnread(t *testing.T) {

	p := NewBytesParser([]byte("ab"))

	c, _ := p.Byte()
	if c != 'a' {
		t.Fatal("read failed: a")
	}
	p.UnreadByte()
	c, _ = p.Byte()
	if c != 'a' {
		t.Fatal("Unread failed: a")
	}
	c, _ = p.Byte()
	if c != 'b' {
		t.Fatal("read failed: b")
	}
	c, _ = p.Byte()
	if c != 0 {
		t.Fatal("read failed: EOS")
	}

	p.UnreadByte()
	p.UnreadByte()
	c, _ = p.Byte()
	if c != 'b' {
		t.Fatal("Unread failed: b")
	}
	c, _ = p.Byte()
	if c != 0 {
		t.Fatal("read failed: EOS")
	}
	c, _ = p.Byte()
	if c != 0 {
		t.Fatal("read failed: EOS")
	}
}

// chars.go
// Character classes. Samples.

func TestChars(t *testing.T) {
	if !IsSpaceChar(' ') {
		t.Error("Error in character class")
	}

	if !IsBreakChar('\n') {
		t.Error("Error in character class")
	}

	if !IsBreakChar('\r') {
		t.Error("Error in character class")
	}

	if !IsSpaceChar('\t') {
		t.Error("Error in character class")
	}

	if IsSpaceChar('a') {
		t.Error("Error in character class")
	}

	if !IsTextChar('(') {
		t.Error("Error in character class")
	}

	if IsTextChar('\t') {
		t.Error("Error in character class")
	}

	if !IsTextChar('_') {
		t.Error("Error in character class")
	}

	if !IsDigit('1') {
		t.Error("Error in character class")
	}

	if IsDigit('x') {
		t.Error("Error in character class")
	}

	if IsDigit(-1) {
		t.Error("Error in character class")
	}

	if IsLetter('1') {
		t.Error("Error in character class")
	}

	if IsLetter(-1) {
		t.Error("Error in character class")
	}

	if !IsEndChar(0) {
		t.Error("Error in character class")
	}

	if IsEndChar('\t') {
		t.Error("Error in character class")
	}

	if !isOperatorChar('>') {
		t.Error("Error in character class")
	}

	if isTemplateTextChar('$') {
		t.Error("Error in character class")
	}

	if !isTemplateTextChar(' ') {
		t.Error("Error in character class")
	}

	if isTokenChar('$') {
		t.Error("Error in character class")
	}
}

// graph.go

func TestGet2(t *testing.T) {

	g := FromString("a b")
	n := g.Get("a")
	s := n.Text()
	if s != "b" {
		t.Error("ogdl.Get")
	}

	g = New(nil)
	g.Add("a").Add("b")

	n = g.Get("a")
	s = n.Text()
	if s != "b" {
		t.Error("ogdl.Get")
	}

	g = New("a")
	g.Add("b")

	// Get does not operate on the root node
	n = g.Get("a")
	s = n.Text()
	if s != "" {
		t.Error("ogdl.Get (root node)")
	}

	g = FromString("a\n b\n c")

	n = g.Get("a")
	s = n.Text()
	if s != "b\nc" {
		t.Error("ogdl.Get (root node)")
	}

	g = FromString("a\n b\n c")
	n = g.Get("a[0]")
	s = n.Text()
	if s != "b" {
		t.Error("ogdl.Get", s)
	}

	g = FromString("a b")
	n = g.Get("a[0]")
	s = n.Text()
	if s != "b" {
		t.Error("ogdl.Get")
	}

	// Index
	g = FromString("a\nb")
	n = g.Get("[0]")

	s = n.Text()
	if s != "a" {
		t.Error("Get index", n.Text())
	}
}

// Check correct get chaining
func TestGet3(t *testing.T) {

	// Simple case
	g := FromString("a b c")
	n := g.Get("a").Get("b")
	m := g.Get("a.b")

	t1, _ := g.GetString("a.b")
	t2 := n.String()

	s := n.Text()
	if s != "c" {
		t.Error("Get case 1", n.Text())
	}

	s = m.Text()
	if s != "c" {
		t.Error("Get case 2", m.Text())
	}

	if t1 != "c" {
		t.Error("Get eqv", m.Text())
	}

	if t1 != t2 {
		t.Error("Get eqv", m.Text())
	}

	// Index
	g = FromString("a\n c\n b")
	n = g.Get("a").Get("[0]")
	m = g.Get("a[0]")

	s = n.Text()
	if s != "c" {
		t.Error("Get case 3", n.Text())
	}

	s = m.Text()
	if s != "c" {
		t.Error("Get case 4", m.Text())
	}

	// Selector
}

func TestCopyAndSubstitute(t *testing.T) {
	g := FromString("a b, c d, aa a")

	g2 := New("_")
	g2.Copy(g)

	if !g.Equals(g2) {
		t.Error("copy or equal failed")
	}

	g3 := FromString("x b, c d, aa x")
	g.Substitute("a", "x")
	if !g.Equals(g3) {
		t.Error("copy or equal failed")
	}
}

func TestGetChaining(t *testing.T) {

	g := FromString("a b c")

	n := g.Get("x").String()

	if n != "" {
		t.Error("Get('unknown').String() should return ''")
	}

	i := g.Get("x").Int64()

	if i != 0 {
		t.Error("Get('unknown').Int64() should return 0")
	}

	i = g.Get("x").Int64(-2)

	if i != -2 {
		t.Error("Get('unknown').Int64(default) should return default")
	}

	n = g.Get("a").Get("b").String()
	if n != "c" {
		t.Error("Get chaining error")
	}
}

func TestGet1(t *testing.T) {

	g := FromString("a b c")

	v, _ := g.GetString("a.b")

	if v != "c" {
		t.Error("GetString()", g.Get("a.b").String())
	}

	g2 := g.Get("a.b")
	if g2.String() != "c" {
		t.Error("Get")
	}

	g.Add("n").Add(1)
	g.Add("d").Add(1.0)

	i := g.Get("n").Scalar()
	s := _typeOf(i)
	if s != "int64" || i != int64(1) {
		t.Error("Scalar()")
	}

	f := g.Get("d").Scalar()
	s = _typeOf(f)
	if s != "float64" || f != 1.0 {
		t.Error("Scalar()", f, s)
	}
}

// A null or new graph should return size = 0

func TestNilGraph(t *testing.T) {

	g := New(nil)

	if g.Len() != 0 {
		t.Error("nil node size not 0")
	}

	g = New(nil)

	if g.Len() != 0 {
		t.Error("nil node size not 0")
	}
}

func TestNewGraph(t *testing.T) {

	g := New("a")

	if g.Len() != 0 {
		t.Error("new node size not 0")
	}
}

func TestAddChaining(t *testing.T) {

	g := FromString("a")
	s := g.Show()
	if s != "_\n  a" {
		t.Error("New( ... )")
	}

	g.Add("b").Add("c")
	s = g.Show()
	if s != "_\n  a\n  b\n    c" {
		t.Error("Add chaining")
	}

	g = New(nil).Add("b")
	s = g.Show()
	if s != "b" {
		t.Error("Add after NewGraph")
	}

	g = New(nil)
	g.Add("a").Add("b")
	s = g.Show()
	if s != "_\n  a\n    b" {
		t.Error("Add chaining on NilGraph")
	}
}

func TestGraph_String(t *testing.T) {
	g := New("*")
	s := g.String()
	if len(s) != 0 {
		t.Error("g.String() returns something with a nil node")
	}
}

func TestGraph_Delete(t *testing.T) {

	g := New(nil)

	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)
	g.Add(5)

	if g.Len() != 5 {
		t.Fatal("g.Len()!=5")
	}

	g.DeleteAt(2)

	if g.Len() != 4 {
		t.Fatal("g.Len()!=4")
	}

	g.DeleteAt(5)
	if g.Len() != 4 {
		t.Fatal("g.Len()!=4")
	}

	g.DeleteAt(0)
	if g.Len() != 3 {
		t.Fatal("g.Len()!=3")
	}

	// Delete element == 5 !
	g.Delete(5)
	if g.Len() != 2 {
		t.Fatal("g.Len()!=2")
	}

	if g.Node("5") != nil {
		t.Error("5 not deleted")
	}
}

// Get types

func TestGetTypes(t *testing.T) {

	g := FromString("aa\nab\nbb\naxx\naj\nvv")
	r, _ := g.Find("a[a-b]")

	if r.Len() != 2 || r.Text() != "aa\nab" {
		t.Error("Find(regex)")
	}

	g = FromString("111")
	if g.Int64() != 111 {
		t.Error("Int64")
	}

	g = FromString("111.1")
	if g.Float64() != 111.1 {
		t.Error("Float64")
	}

	g = New(nil)
	g.Add(float32(111.2))
	if g.Float64() != 111.2 {
		t.Error("Float64")
	}

	g = FromString("true")
	if g.Bool() != true {
		t.Error("Bool")
	}

	g = FromString("a 1")
	if i, _ := g.GetInt64("a"); i != 1 {
		t.Error("GetInt64")
	}

	g = FromString("a 1.1")
	if i, _ := g.GetFloat64("a"); i != 1.1 {
		t.Error("GetFloat64")
	}

	g = FromString("a 'false'")
	if i, err := g.GetBool("a"); err != nil || i != false {
		t.Error("GetBool")
	}

	g = FromString("a 'text'")
	if i, err := g.GetBytes("a"); err != nil || len(i) != 4 {
		t.Error("GetBytes", len(i))
	}
}

func TestIsInteger(t *testing.T) {
	ss := [...]string{"-1", "2", "9.1", " 14", " - 1", " -1 ", "a", "3a", ""}
	rr := [...]bool{true, true, false, true, false, true, false, false, false}

	for i, s := range ss {
		b := isInteger(s)
		if b != rr[i] {
			t.Error("IsInteger() failed")
		}
	}
}

// interface conversion to native types

func TestI2string(t *testing.T) {

	var i interface{}

	i = "2.2"

	s := _string(i)
	ty := reflect.TypeOf(s).String()

	if ty != "string" {
		t.Error("should return a string")
	}

	i = 1.1

	s = _string(i)
	ty = reflect.TypeOf(s).String()

	if ty != "string" {
		t.Error("should return a string")
	}
}

// function.go

func TestFunction3(ts *testing.T) {
	// Context
	g := New(nil)

	t := NewTemplate("$(R.y='h')$(R.x0='user')")
	t.Process(g)

	t = NewTemplate("$R")
	b := t.Process(g)

	if string(b) != "y\n  h\nx0\n  user" {
		ts.Error("Template processing of variable", string(b))
	}

	t = NewTemplate("$T(R)")
	b = t.Process(g)

	/*

		println(t.Show())
		b := t.Process(g)

		if string(b) != "Title: A nice title" {
			ts.Error("function T", string(b))
		} */
}

type Math struct {
}

func (*Math) Sin(x float64) float64 {
	return math.Sin(x)
}

func Sin(x float64) float64 {
	return math.Sin(x)
}

func TestFunction2b(t *testing.T) {

	g := New(nil)
	f := g.Add("math")
	f.Add(&Math{})

	path := NewPath("math.Sin(1.0)")

	fmt.Printf("%s\n", path.Show())

	i, _ := g.Eval(path)
	s := _typeOf(i)
	v := _string(i)[:10]

	if s != "float64" || v != "0.84147098" {
		t.Error("math.Sin()", s, v, i)
	}
}

func TestFunction2c(t *testing.T) {

	g := New(nil)
	f := g.Add("Sin")
	f.Add(Sin)

	path := NewPath("Sin(1.0)")

	i, _ := g.Eval(path)
	s := _typeOf(i)
	v := _string(i)[:10]

	if s != "float64" || v != "0.84147098" {
		t.Error("Sin()", s, v)
	}
}

// log.go

func TestLog(t *testing.T) {

	file := "/tmp/log.gb"

	log, _ := OpenLog(file)

	g := FromString("a b\nc\nd")
	b := g.Binary()

	n := log.Add(g)
	m := log.Add(g)
	o := log.AddBinary(b)

	if n != 0 || m != 16 || o != 32 {
		t.Error("log.Add", o)
	}

	g2, n2, _ := log.Get(0)
	g3, n3, _ := log.Get(n2)
	b2, _, _ := log.GetBinary(n3)

	g4 := FromBinary(b2)

	if !g.Equals(g2) {
		t.Error("n!=n2")
	}

	if !g.Equals(g3) {
		t.Error("n!=n3")
	}

	if !g.Equals(g4) {
		t.Error("n!=n3")
	}

	log.Sync()

	log.Close()

	os.Remove(file)
}

// -------------------------------------------------------------------------
// EXAMPLES
// -------------------------------------------------------------------------

func ExampleGraph_Set() {

	g := FromString("a b c")
	g.Set("a.b", "d")

	fmt.Println(g.Text())

	// Output:
	// a
	//   b
	//     d
}

func ExampleGraph_Set_index() {

	g := FromString("a b c")
	g.Set("a[1]", "d")

	fmt.Println(g.Text())

	// Output:
	// a
	//   b
	//     c
	//   d
}

func ExampleGraph_Set_a() {

	g := New(nil)

	g.Add("R").Add("b")
	r := g.Node("R")
	r.Set("id", "1")

	fmt.Println(g.Text())
	// Output:
	// R
	//   b
	//   id
	//     1
}

func ExampleGraph_Check() {

	schema := FromString("a !int\nb !string\nc !float\nd !bool")
	g := FromString("a 1\nb s\nc 1.0\nd true")

	b, message := schema.Check(g)
	fmt.Println(b, message)
	// Output:
	// true
}
