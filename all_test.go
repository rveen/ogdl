package ogdl

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

// path.go

func TestPath(t *testing.T) {

	p := NewPath("a")

	if p.Len() != 1 {
		t.Error("size != 1")
	}

	p = NewPath("a.b")
	s := p.Show()
	if s != "!p\n  a\n  b" {
		t.Error("Path, index", s)
	}

	p = NewPath("[0]")

	s = p.Show()
	if s != "!p\n  !i\n    0" {
		t.Error("Path, index", s)
	}

	p = NewPath("a[1]")

	s = p.Show()
	if s != "!p\n  a\n  !i\n    1" {
		t.Error("Path, index")
	}

	// Function arguments vs. expression elements

	p = NewPath("a(b)")

	s = p.Show()
	if s != "!p\n  a\n  !a\n    !e\n      !p\n        b" {
		t.Error("Path, index", s)
	}
}

func TestExpressionInPath(t *testing.T) {
	p := NewPath("a.(b)")

	s := p.Show()
	if s != "!p\n  a\n  !g\n    !e\n      !p\n        b" {
		t.Error("Path, index", s)
	}

	// Todo: a() check that it can tolerate zero arguments
}

func TestFlowSyntaxInPath(t *testing.T) {
	p := NewPath("(a, b)")
	s := p.Show()
	if s != "!p\n  !g\n    !e\n      !p\n        a\n    !e\n      !p\n        b" {
		t.Error("Flow syntax in path", s)
	}

	p = NewPath("(a b)")
	s = p.Show()
	fmt.Printf("%s\n", s)
	if s != "!p\n  !g\n    !e\n      !p\n        a\n      !e\n        !p\n          b" {
		t.Error("Flow syntax in path", s)
	}
}

// binary.go

func TestBinParser1(t *testing.T) {

	// newVarInt
	b := newVarInt(0x3fff)
	p := newBytesBinParser(b)
	i := p.varInt()
	if i != 0x3fff && len(b) != 2 {
		t.Error("varInt 0x3fff")
	}

	b = newVarInt(0x4000)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 0x4000 && len(b) != 3 {
		t.Error("varInt 0x4000")
	}

	b = newVarInt(0x1fffff)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 0x1fffff && len(b) != 4 {
		t.Error("varInt 0x1fffff", i)
	}

	b = newVarInt(0xfffffff)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 0xfffffff && len(b) != 4 {
		t.Error("varInt 0xfffffff", i)
	}

	b = newVarInt(127)
	p = newBytesBinParser(b)
	i = p.varInt()
	if i != 127 && len(b) != 1 {
		t.Error("varInt 127")
	}

	b = newVarInt(-1)
	if b != nil {
		t.Error("newVarInt -1")
	}
	b = newVarInt(0x10000000)
	if b != nil {
		t.Error("newVarInt to high")
	}

	// force incorrect header
	h := []byte{1, 'G', 0}
	h2 := []byte{0, 'G', 0}
	h3 := []byte{1, 'H', 0}
	h4 := []byte{1, 'G', 1}

	p = newBytesBinParser(h)
	if !p.header() {
		t.Error("header")
	}
	p = newBytesBinParser(h2)
	if p.header() {
		t.Error("header")
	}
	p = newBytesBinParser(h3)
	if p.header() {
		t.Error("header")
	}
	p = newBytesBinParser(h4)
	if p.header() {
		t.Error("header")
	}
}

func TestBinParser2(t *testing.T) {

	s := "a"

	p := newBinParser(bytes.NewReader([]byte(s)))

	c := p.read()
	if c != 'a' {
		t.Error("read error")
	}

	p.unread()

	c = p.read()
	if c != 'a' {
		t.Error("unread error")
	}

	c = p.read()
	if c != -1 {
		t.Error("EOS read error")
	}
}

func TestBinParser3(t *testing.T) {

	r := []byte{1, 'G', 0, 1, 'a', 0, 2, 'b', 0, 0}

	// Starting from a NilGraph
	g := New()
	b := g.Binary()

	if len(b) != 4 {
		t.Error("Binary() on NilGraph")
	}

	// Starting from a NilGraph
	g = New()
	g.Add("a").Add("b")
	b = g.Binary()

	if string(r) != string(b) {

		for i := 0; i < len(b); i++ {
			println(b[i])
		}

		t.Error("Binary() failed")
	}

	var nul *Graph
	b = nul.Binary()
	if b != nil {
		t.Error("Binary nil failed")
	}

	g = FromBinary(r)
	// g = g.Out[0]

	if g.Len() != 1 {
		t.Error("BinParse() failed")
	}
	if g.String() != "a" {
		t.Error("BinParse() failed")
	}
}

func TestBinParser4(t *testing.T) {

	r := []byte{1, 'G', 0 /*lev*/, 1 /*bin*/, 1 /* len */, 1, 0x55 /*end bin */, 0, 0}

	g := FromBinary(r)
	//g = g.Out[0]

	if g.Len() != 1 {
		t.Error("BinParse() failed")
	}
	if g.String() != "U" {
		t.Error("BinParse() failed")
	}
}

// parser.go

func TestParser00(t *testing.T) {
	p := FromString("a")

	if p.This != nil {
		t.Error("parse should always return a nil root")
	}

	if p.Text() != "a" {
		t.Error("e1", p.Text())
	}

	p = FromString("a\nb")

	if p.This != nil {
		t.Error("parse should always return a nil root")
	}

	if p.Text() != "a\nb" {
		t.Error("e2", p.Show())
	}
}

func TestParser0(t *testing.T) {

	p := NewParser(strings.NewReader("a b"))
	p.Ogdl()
	s := p.Graph().Text()
	if s != "a\n  b" {
		t.Error("Parser0")
	}
}

var textIn = [...]string{
	"a",
	"\na",
	"a\n",
	"a     ",
	"a     \n",
	"a\nb",
	"x\r\ny",
}

var textOut = [...]string{
	"a",
	"a",
	"a",
	"a",
	"a",
	"a\nb",
	"x\ny",
}

func TestParser1(t *testing.T) {

	for i := 0; i < len(textIn); i++ {
		g := FromString(textIn[i])
		if g.Text() != textOut[i] {
			t.Error("Parser error")
		}
	}
}

// line level resolution (need asserts)
// TODO:
/*
func Test_Level(t *testing.T) {

	p := NewStringParser("")

	// First time any n will return level 0
	p.setLevel(0, 0)
	pr(p)

	p.setLevel(1, 2)
	pr(p)

	p.setLevel(3, 4)
	pr(p)
}

func pr(p *Parser) {
	for i := 0; i < 10; i++ {
		print(p.getLevel(i), " ")
	}
	println("")

	for i := 0; i < 5; i++ {
		print(p.ind[i], " ")
	}
	println("\n--")
} */

// Special cases

func TestParser2(t *testing.T) {

	g := FromString(" a")
	if g.Text() != "a" {
		t.Error("e1", g.Text())
	}

	g = FromString("a b c d")
	if g.Text() == "a\n  b\n    c\n    d" {
		t.Error("e2")
	}
}

// Blocks

func TestParseBlock1(t *testing.T) {
	p := NewBytesParser([]byte("a \\\n  b\n  c"))
	p.String()
	p.Space()
	p.ev.Inc()
	s, ok := p.Block(0)
	if !ok || s != "b\nc" {
		t.Error("block")
	}
}

func TestParseBlock2(t *testing.T) {
	g := FromString("a \\\n  b c")
	if g.Text() != "a\n \"b c\"" {
		t.Error()
	}
}

func TestBlockPrint(t *testing.T) {
	g := FromString("a \\\n  b\n  c")

	// Blocks are currently printed back as quoted strings
	if g.Text() != "a\n \"b\n  c\"" {
		t.Error("block print\n", g.Text())
	}
}

// Comments

func TestComment(t *testing.T) {

	g := FromString("# comment")
	if g.Text() != "" {
		t.Error("comment 1:", g.Text())
	}

	g = FromString("# comment\nnot#acomment")
	if g.Text() != "not#acomment" {
		t.Error("comment 2:", g.Text())
	}
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
	if !isSpaceChar(' ') {
		t.Error("Error in character class")
	}

	if !isBreakChar('\n') {
		t.Error("Error in character class")
	}

	if !isBreakChar('\r') {
		t.Error("Error in character class")
	}

	if !isSpaceChar('\t') {
		t.Error("Error in character class")
	}

	if isSpaceChar('a') {
		t.Error("Error in character class")
	}

	if !isTextChar('(') {
		t.Error("Error in character class")
	}

	if isTextChar('\t') {
		t.Error("Error in character class")
	}

	if !isTextChar('_') {
		t.Error("Error in character class")
	}

	if !isDigit('1') {
		t.Error("Error in character class")
	}

	if isDigit('x') {
		t.Error("Error in character class")
	}

	if isDigit(-1) {
		t.Error("Error in character class")
	}

	if isLetter('1') {
		t.Error("Error in character class")
	}

	if isLetter(-1) {
		t.Error("Error in character class")
	}

	if !isEndChar(0) {
		t.Error("Error in character class")
	}

	if isEndChar('\t') {
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

	g = New()
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

	g2 := New()
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

	g := New()

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

	g = New().Add("b")
	s = g.Show()
	if s != "b" {
		t.Error("Add after NewGraph")
	}

	g = New()
	g.Add("a").Add("b")
	s = g.Show()
	if s != "_\n  a\n    b" {
		t.Error("Add chaining on NilGraph")
	}
}

func TestGraph_String(t *testing.T) {
	g := New()
	s := g.String()
	if len(s) != 0 {
		t.Error("g.String() returns something with a nil node")
	}
}

func TestGraph_Delete(t *testing.T) {

	g := New()

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

func TestEvalPath1(t *testing.T) {

	// Create a path and check it
	path := NewPath("a")
	s := path.Show()
	if s != "!p\n  a" {
		t.Error("NewPath", s)
	}

	g := New()
	g.Add("a").Add("b")

	i, _ := g.evalPath(path)

	if _string(i) != "b" {
		t.Error("EvalPath", _show(i))
	}

	g = New()
	g.Add("a").Add(1)

	i, _ = g.evalPath(path)
	s = _typeOf(i)
	if i != 1 || s != "int" {
		t.Error("EvalPath 1", _show(i), _typeOf(i))
	}

	g = New()
	g.Add("a").Add("id").Add("100")
	i, _ = g.evalPath(path)

	if _text(i) != "id\n  100" || _typeOf(i) != "*ogdl.Graph" {
		println(_show(i), _typeOf(i))
		t.Error()
	}
}

func TestEvalPath2(t *testing.T) {

	g := FromString("a\n b\n  c\n  d")

	p := NewPath("a")
	i, _ := g.evalPath(p)
	if _show(i) != "_\n  b\n    c\n    d" {
		t.Error("e1", _show(i))
	}

	p = NewPath("a[0]")
	i, _ = g.evalPath(p)
	if _show(i) != "_\n  b\n    c\n    d" {
		t.Error("e2", _show(i))
	}

}

func TestEvalPath3(t *testing.T) {
	g := FromString("a\n b 1\n b 2")

	p := NewPath("a.b[0]")

	i, _ := g.evalPath(p)

	if _string(i) != "1" {
		t.Error("e1", _show(i))
	}

	p = NewPath("a.b")

	i, _ = g.evalPath(p)

	if _string(i) != "1" {
		t.Error("e2", _show(i))
	}

	p = NewPath("a.b{1}")

	i, _ = g.evalPath(p)

	if _string(i) != "2" {
		t.Error("e3", _text(i))
	}

	p = NewPath("a.b{}")

	i, _ = g.evalPath(p)

	if _text(i) != "1\n2" {
		t.Error("e4", _text(i))
	}
}

func TestEvalScalar(t *testing.T) {

	g := New()
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

// TestEvalArgOfGraph tests the following situation:
//
//    $a(b)
//
// where a is a graph and b is a variable containing a path
//
func TestEvalArgOfGraph(t *testing.T) {

	g := New()
	g.Add("a").Add("c").Add(int64(1))
	g.Add("b").Add("c")

	p := NewPath("a.(b)")
	fmt.Printf("%s\n", p.Show())
	r, _ := g.Eval(p)

	ty := _typeOf(r)

	if ty != "int64" || r != int64(1) {
		t.Errorf("a(b): %s %v", ty, r)
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
	g = New()
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

	g := New()
	g.Add("a").Add(1)

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

	g = New()
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

// Templates

func TestTemplate1(ts *testing.T) {
	// Context
	g := New()
	g.Add("b").Add(1)

	t := NewTemplate("a $b")

	s := t.Process(g)

	if string(s) != "a 1" {
		ts.Error("template", string(s))
	}
}

func TestTemplateOperatorConfusion(ts *testing.T) {
	// Context
	g := New()
	g.Add("b").Add(1)

	t := NewTemplate("$(a='/') $a $b")

	s := t.Process(g)

	if string(s) != " / 1" {
		ts.Error("template", string(s))
	}
}

func TestTemplateIfEmptyString(ts *testing.T) {
	// Context
	g := New()
	g.Add("b").Add("")

	t := NewTemplate("$if(b=='') text $end")

	s := t.Process(g)

	if string(s) != " text " {
		ts.Error("template", string(s))
	}
}

func TestTemplateIf0(ts *testing.T) {
	t := NewTemplate("a $if('true') x $end")

	fmt.Printf("%s\n", t.Show())
}

func TestTemplateIf(ts *testing.T) {
	// Context
	g := New()

	t := NewTemplate("$if('false') a $else b $end")

	fmt.Printf("%s\n", t.Show())

	s := t.Process(g)
	if string(s) != " b " {
		ts.Error("template")
	}
	/*
		t = NewTemplate("$if('true') a $else b $end")
		s = t.Process(g)
		if string(s) != " a " {
			ts.Error("template")
		} */
}

func TestTemplateFor(ts *testing.T) {
	// Context
	g := New()
	c := g.Add("b")
	c.Add(1)
	c.Add(2)

	t := NewTemplate("$b\n---\n$for(a,b) [$a] $end")

	s := t.Process(g)
	if string(s) != "1\n2\n---\n [1]  [2] " {
		ts.Errorf("template\n%s", string(s))
	}

	// The variable
	g = FromString("result\n  item id 0\n  item id 1")

	t = NewTemplate("$for(a,result) $a.item.id $end")

	s = t.Process(g)
	if string(s) != " 0  1 " {
		ts.Error("template: " + string(s))
		println(g.Show())
	}
}

// function.go

func TestFunction3(ts *testing.T) {
	// Context
	g := New()

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

	g := New()
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

	g := New()
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

	g := New()

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

func ExampleGraph_Get() {
	g := FromString("a\n b 1\n c 2\n b 3")
	fmt.Println(g.Get("a.b{0}").Text())
	fmt.Println(g.Get("a.b{1}").Text())
	fmt.Println("---")
	fmt.Println(g.Get("a.b{}").Text())
	fmt.Println("---")
	fmt.Println(g.Get("a[0]").Text())
	// Output:
	// 1
	// 3
	// ---
	// 1
	// 3
	// ---
	// b
	//   1
}

func ExampleNewTemplate() {
	p := NewTemplate("Hello, $user")

	g := New()
	g.Add("user").Add("Jenny")

	fmt.Println(string(p.Process(g)))
	// Output:
	// Hello, Jenny
}

func ExampleNewExpression() {
	e := NewExpression("1-2+3")
	g := New()
	i, _ := g.Eval(e)

	fmt.Println(i)
	// Output:
	// 2
}

func ExampleGraph_Check() {

	schema := FromString("a !int\nb !string\nc !float\nd !bool")
	g := FromString("a 1\nb s\nc 1.0\nd true")

	b, message := schema.Check(g)
	fmt.Println(b, message)
	// Output:
	// true
}

func ExampleGraph_Eval() {
	g := New()
	g.Add("a").Add(4)
	g.Add("b").Add("4")
	e := NewExpression("a+3")
	e2 := NewExpression("b+3")
	fmt.Println(g.Eval(e))
	fmt.Println(g.Eval(e2))
	// Output:
	// 7 <nil>
	// 43 <nil>
}
