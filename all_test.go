package ogdl

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"bytes"
	"os"
)

// path.go

func TestPath1(t *testing.T) {

	p := NewPath("a")

	if p.Len() != 1 {
		t.Error("size != 1")
	}
}

func TestPath2(t *testing.T) {
	p := NewPath("a.b")

	if p.Len() != 2 {
		t.Error("size != 2")
	}
}

// binary.go

func TestBinParser1(t *testing.T) {

    // newVarInt
	b := newVarInt(0x3fff)
	p := NewBytesBinParser(b)
	i := p.varInt()
	if i != 0x3fff && len(b) != 2 {
		t.Error("varInt 0x3fff")
	}

	b = newVarInt(0x4000)
	p = NewBytesBinParser(b)
	i = p.varInt()
	if i != 0x4000 && len(b) != 3 {
		t.Error("varInt 0x4000")
	}
	
	b = newVarInt(0x1fffff)
	p = NewBytesBinParser(b)
	i = p.varInt()
	if i != 0x1fffff && len(b) != 4 {
		t.Error("varInt 0x1fffff",i)
	}
	
	b = newVarInt(0xfffffff)
	p = NewBytesBinParser(b)
	i = p.varInt()
	if i != 0xfffffff && len(b) != 4 {
		t.Error("varInt 0xfffffff",i)
	}

	b = newVarInt(127)
	p = NewBytesBinParser(b)
	i = p.varInt()
	if i != 127 && len(b) != 1 {
		t.Error("varInt 127")
	}
	
	b = newVarInt(-1)
	if b!=nil {
	    t.Error("newVarInt -1")
	}
	b = newVarInt(0x10000000)
	if b!=nil {
	    t.Error("newVarInt to high")
	}
	
	// force incorrect header
	h := []byte{ 1, 'G', 0 }
	h2 := []byte{ 0, 'G', 0 }
	h3 := []byte{ 1, 'H', 0 }
	h4 := []byte{ 1, 'G', 1 }
	
	p = NewBytesBinParser(h)
	if !p.header() {
	    t.Error("header")
	}
	p = NewBytesBinParser(h2)
	if p.header() {
	    t.Error("header")
	}
	p = NewBytesBinParser(h3)
	if p.header() {
	    t.Error("header")
	}
	p = NewBytesBinParser(h4)
	if p.header() {
	    t.Error("header")
	}
}

func TestBinParser2 (t *testing.T) {

    s := "a"
    
    p := NewBinParser(bytes.NewReader([]byte(s)))

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

func TestBinParser3 (t *testing.T) {
    
    r := []byte{ 1, 'G', 0, 1, 'a', 0, 2, 'b', 0, 0 } 
    
    g := NewGraph("a")
    g.Add("b")
    b := g.Binary()
    
    if string(r)!=string(b) {
        t.Error("Binary() failed")
    }
    
    var nul *Graph
    b = nul.Binary()
    if b != nil {
        t.Error("Binary nil failed")
    }
    
    g = BinParse(r)
    
    if g.Len() != 1 {
        t.Error("BinParse() failed")
    }
    if g.String() != "a" {
        t.Error("BinParse() failed")
    }
}

func TestBinParser4 (t *testing.T) {
    
    r := []byte{ 1, 'G', 0, /*lev*/ 1, /*bin*/ 1, /* len */ 1, 0x55, /*end bin */ 0, 0 } 
    
    g := BinParse(r)
    
    if g.Len() != 0 {
        t.Error("BinParse() failed")
    }
    if g.String() != "U" {
        t.Error("BinParse() failed")
    }
}
// parser.go

// This array should hold all known cases
// that the parser may encounter.

var text = [...]string{
	// Short documents
	"a",
	"\na",
	"a\n",
	" a",
	"a     ",
	"a     \n",
	"a\nb",
	"x\r\ny",
	"x \r \n y",
	// Blocks
	"a \\\n  b",
	"a \\ \n  b", // Space between \ and end of line. Nasty.
	"a \\\n  b\n",
	"a \\\n  b\n  c",
	// Groups
	"a (b c) d", // Should trigger an error.
	"a (b c), d",
	// Quoted

}

func TestParser1(t *testing.T) {

/*
	for i := 0; i < len(text); i++ {
		g := ParseString(text[i])
		println(g.Text())
	}*/
	
	n := ParseString(" a")
	println(n.Text())
}

func TestUnread(t *testing.T) {

	p := NewStringParser("ab")

	c := p.Read()
	if c != 'a' {
		t.Fatal("read failed: a")
	}
	p.Unread()
	c = p.Read()
	if c != 'a' {
		t.Fatal("Unread failed: a")
	}
	c = p.Read()
	if c != 'b' {
		t.Fatal("read failed: b")
	}
	c = p.Read()
	if c != 0 {
		t.Fatal("read failed: EOS")
	}

	p.Unread()
	p.Unread()
	c = p.Read()
	if c != 'b' {
		t.Fatal("Unread failed: b")
	}
	c = p.Read()
	if c != 0 {
		t.Fatal("read failed: EOS")
	}
	c = p.Read()
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

	if !IsSpaceChar('\t') {
		t.Error("Error in character class")
	}

	if IsSpaceChar('a') {
		t.Error("Error in character class")
	}

	if IsTextChar('(') {
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

	if IsOperatorChar(-1) {
		t.Error("Error in character class")
	}

	if !IsOperatorChar('>') {
		t.Error("Error in character class")
	}

	if IsTemplateTextChar('$') {
		t.Error("Error in character class")
	}

	if !IsTemplateTextChar(' ') {
		t.Error("Error in character class")
	}

	if IsTokenChar('$') {
		t.Error("Error in character class")
	}
}

// graph.go

func TestCopyAndSubstitute ( t *testing.T ) {
    g := ParseString("a b, c d, aa a")
    
    g2 := NilGraph()
    g2.Copy(g)
    
    if !g.Equal(g2) {
        t.Error("copy or equal failed")
    }
    
    g3 := ParseString("x b, c d, aa x")
    g.Substitute("a","x")
    if !g.Equal(g3) {
        t.Error("copy or equal failed")
    }
}

func TestGet(t *testing.T) {

	g := NewGraph("a")
	g.Add("b").Add("c")

	v, _ := g.GetString("a")
	s := reflect.TypeOf(v).String()
	fmt.Printf("%s / %v\n", s, v)

	g2 := g.Get("a.b")
	s = reflect.TypeOf(g2).String()
	fmt.Printf("%s / %v\n", s, g2)

	g.Add("n").Add(1)
	g.Add("d").Add(1.0)

	i := g.Get("a.n").Scalar()
	s = reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)
	i = g.Get("a.d").Scalar()
	s = reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)
}

// A null or new graph should return size = 0

func TestNilGraph(t *testing.T) {

	g := NilGraph()

	if g.Len() != 0 {
		t.Error("nil node size not 0")
	}

	if !g.IsNil() {
		t.Error("IsNil() incorrect")
	}

	g = NewGraph(nil)

	if g.Len() != 0 {
		t.Error("nil node size not 0")
	}

	if !g.IsNil() {
		t.Error("IsNil() incorrect")
	}

}

func TestNewGraph(t *testing.T) {

	g := NewGraph("a")

	if g.Len() != 0 {
		t.Error("new node size not 0")
	}

	if g.IsNil() {
		t.Error("false IsNil()")
	}
}

func TestDepth(t *testing.T) {
	g := NewGraph("a")

	if g.Depth() != 0 {
		t.Error("g.Depth() != 0")
	}

	g.Add("b")

	if g.Depth() != 1 {
		t.Error("g.Depth() != 1")
	}

	n := g.Add("c")

	if g.Depth() != 1 {
		t.Error("g.Depth() != 1")
	}

	nn := n.Add("d")

	if g.Depth() != 2 {
		t.Error("g.Depth() != 2")
	}

	nn.Add("e")

	if g.Depth() != 3 {
		t.Error("g.Depth() != 3")
	}
}

func TestGraph_String(t *testing.T) {
	g := NilGraph()
	s := g.String()
	if len(s) != 0 {
		t.Error("g.String() returns something with a nil node")
	}
}

func TestGraph_Delete(t *testing.T) {

	g := NilGraph()

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
    if s!="int" || i!=1 {
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
	
	b = compare(2,1,'>')
	if !b {
		t.Error("compare >")
	}
	b = compare(2.0,1.0,'>')
	if !b {
		t.Error("compare >")
	}
	
	// <=
	b = compare(1,2,'-')
	if !b {
		t.Error("compare <=")
	}
	b = compare(2,2,'-')
	if !b {
		t.Error("compare <=")
	}
	b = compare(2.0,2.0,'-')
	if !b {
		t.Error("compare <=")
	}
	
	// >=
	b = compare(2,1,'+')
	if !b {
		t.Error("compare >=")
	}
	b = compare(2,2,'+')
	if !b {
		t.Error("compare >=")
	}
	b = compare(2.0,2.0,'+')
	if !b {
		t.Error("compare >=")
	}
	
	// Cannot compare non numbers
	b = compare("a",1,'=')
	if b {
		t.Error("compare a string")
	}
	b = compare(1,"a",'=')
	if b {
		t.Error("compare a string")
	}
	
	// !
	b = compare(1,2,'!')
	if !b {
	    t.Error("compare !=")
	}
	b = compare(1,1,'!')
	if b {
	    t.Error("compare !=")
	}
	b = compare(1.0,2.0,'!')
	if !b {
	    t.Error("compare !=")
	}
	b = compare(1.0,1.0,'!')
	if b {
	    t.Error("compare !=")
	}
}

func TestEvalPath(t *testing.T) {

	g := NilGraph()
	g.Add("a").Add(1)

	p := NewPath("a")

	i := g.Eval(p)
	s := reflect.TypeOf(i).String()
	if i!=1 || s!="int" {
	    t.Error("EvalPath 1")
	}

    // Creating a nil root or not should not make a difference
	g = NewGraph("a")
	g.Add(1)
	i = g.Eval(p)
	s = reflect.TypeOf(i).String()
	if i!=1 || s!="int" {
	    t.Error("EvalPath 2")
	}
}

func TestEvalPath_Index(t *testing.T) {
    g := ParseString("a (b 1, b 2)")

    p := NewPath("a.b[0]")
    
    i := g.EvalPath(p)
    
    if _string(i) != "1" {
        t.Error("EvalPath_Index")
    } 
    
    p = NewPath("a.b{1}")
    
    i = g.EvalPath(p)
    
    if _string(i) != "2" {
        t.Error("EvalPath_Selector")
    } 
    
    p = NewPath("a.b{}")
    
    i = g.EvalPath(p)

    if _text(i) != "1\n2" {
        t.Error("EvalPath_Selector",_text(i))
    } 
}

func TestEvalScalar(t *testing.T) {

	g := NilGraph()
	p := NewGraph("1")

	// constants
	i := g.Eval(p)
	s := reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)

	p = NewGraph("1.1")
	i = g.Eval(p)
	s = reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)

	p = NewGraph(1.1)
	i = g.Eval(p)
	s = reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)

	p = NewGraph('c')
	i = g.Eval(p)
	s = reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)

	p = NewGraph("true")
	i = g.Eval(p)
	s = reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)

	p = NewGraph("2")
	i = g.EvalExpression(p)
	s = reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)

	p = NewGraph(true)
	i = g.Eval(p)
	s = reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)

	p = NewGraph("!e")
	pp := p.Add("+")
	pp.Add(1)
	pp.Add(2)

	i = g.Eval(p)
	s = reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)

	p = NewGraph("!e")
	pp = p.Add("+")
	pp.Add(1)
	pp.Add(2.2)

	i = g.Eval(p)
	s = reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)
}

func TestEvalBool1(t *testing.T) {

	g := NilGraph()
	g.Add("a").Add(1)

	p := NewExpression("a==1")
	println(p.Text())
	r := g.EvalBool(p)
	fmt.Printf("a==1: %v\n", r)
}

// TestEvalArgOfGraph tests the following situation:
//
//    $a(b)
//
// where a is a graph and b is a variable containing a path
//
func TestEvalArgOfGraph(t *testing.T) {

	g := NilGraph()
	g.Add("a").Add("c").Add(1)
	g.Add("b").Add("c")

	p := NewPath("a(b)")
	println(p.Text())
	r := g.Eval(p)
	fmt.Printf("a(b): %v\n", r)
}

func TestEvalExpression(t *testing.T) {

	// The context
	g := NewGraph("a")
	g.Add("b")

	// 'a' is a string constant 
	p := NewExpression("a=='b'")
	r := g.Eval(p)
	
	if _typeOf(r) != "bool" || r!=true {
	     t.Error("a=='b'")
	}
	
	p = NewExpression("'a'!='b'")
	r = g.Eval(p)
	if _typeOf(r) != "bool" || r!=true {
	     t.Error("'a'!='b'")
	}
	
	p = NewExpression("2>1")
	r = g.Eval(p)
	if _typeOf(r) != "bool" || r!=true {
	     t.Error("'a'!='b'")
	}
	p = NewExpression("2>=2")
	r = g.Eval(p)
	if _typeOf(r) != "bool" || r!=true {
	     t.Error("'a'!='b'")
	}
	p = NewExpression("1<2")
	r = g.Eval(p)
	if _typeOf(r) != "bool" || r!=true {
	     t.Error("'a'!='b'")
	}
	p = NewExpression("1<=2")
	r = g.Eval(p)
	if _typeOf(r) != "bool" || r!=true {
	     t.Error("'a'!='b'")
	}
	p = NewExpression("1<0")
	r = g.Eval(p)
	if _typeOf(r) != "bool" || r!=false {
	     t.Error("'a'!='b'")
	}
	
	// logic
	e := "'false' || 'true'"
	p = NewExpression(e)
	r = g.Eval(p)
	if _typeOf(r) != "bool" || r!=true {
	     t.Error(e)
	}
	e = "'true' && 'true'"
	p = NewExpression(e)
	r = g.Eval(p)
	if _typeOf(r) != "bool" || r!=true {
	     t.Error(e)
	}
	
	// Assing
	g = NilGraph()
	e = "a=1"
	p = NewExpression(e)
	g.Eval(p)
	
	if i,_ := g.GetInt64("a"); i != 1 {
	    t.Error(e,_typeOf(g),_text(g))
	}
	
	e = "a+=12"
	p = NewExpression(e)
	g.Eval(p)
	
	if i,_ := g.GetInt64("a"); i != 13 {
	    t.Error(e,_typeOf(g),_text(g))
	}
	
	e = "a-=1"
	p = NewExpression(e)
	g.Eval(p)
	
	if i,_ := g.GetInt64("a"); i != 12 {
	    t.Error(e,_typeOf(g),_text(g))
	}
	
	e = "a*=2"
	p = NewExpression(e)
	g.Eval(p)
	
	if i,_ := g.GetInt64("a"); i != 24 {
	    t.Error(e,_typeOf(g),_text(g))
	}
	
	e = "a/=4"
	p = NewExpression(e)
	g.Eval(p)
	
	if i,_ := g.GetInt64("a"); i != 6 {
	    t.Error(e,_typeOf(g),_text(g))
	}
	
	e = "a%=4"
	p = NewExpression(e)
	g.Eval(p)
	
	if i,_ := g.GetInt64("a"); i != 2 {
	    t.Error(e,_typeOf(g),_text(g))
	}
	
	// b non existent
	e = "b+=1"
	p = NewExpression(e)
	g.Eval(p)
	
	if i,_ := g.GetInt64("b"); i != 1 {
	    t.Error(e,_typeOf(g),_text(g))
	}
	
	// c non existent
	e = "c-=1"
	p = NewExpression(e)
	g.Eval(p)
	
	if i,_ := g.GetInt64("c"); i != -1 {
	    t.Error(e,_typeOf(g),_text(g))
	}
	
	// d non existent
	e = "d*=1"
	p = NewExpression(e)
	g.Eval(p)
	
	if i,_ := g.GetInt64("d"); i != 0 {
	    t.Error(e,_typeOf(g),_text(g))
	}
	
	// e non existent
	e = "e/=1"
	p = NewExpression(e)
	g.Eval(p)
	
	if _,err := g.GetInt64("e"); err==nil {
	    t.Error(e,_typeOf(g),_text(g))
	}
	
	e = "1+2"
	p = NewExpression(e)
	r = g.Eval(p)
	
	if r != int64(3) {
	    t.Error(e,_typeOf(r),_text(r))
	}
	
	e = "2-4"
	p = NewExpression(e)
	r = g.Eval(p)
	
	if r != int64(-2) {
	    t.Error(e,_typeOf(r),_text(r))
	}
	
	e = "10*3"
	p = NewExpression(e)
	r = g.Eval(p)
	
	if r != int64(30) {
	    t.Error(e,_typeOf(r),_text(r))
	}
	
	e = "10/3"
	p = NewExpression(e)
	r = g.Eval(p)
	
	if r != int64(3) {
	    t.Error(e,_typeOf(r),_text(r))
	}
	
	e = "10%3"
	p = NewExpression(e)
	r = g.Eval(p)
	
	if r != int64(1) {
	    t.Error(e,_typeOf(r),_text(r))
	}
}

func TestEvalBool2(t *testing.T) {

	// The context
	g := NilGraph()
	g.Add("a").Add(1)

	p := NewExpression("1=='1'")
	r := g.EvalBool(p)
	fmt.Printf("%v\n", r)

	// true is a reserved word ? -----------------
	p = NewExpression("true")
	println(p.Text())
	i := g.EvalBool(p)
	fmt.Printf("%v\n", i)

	// -----------------
	p = NewExpression("'true'")
	r = g.EvalBool(p)
	fmt.Printf("%v\n", r)
}



// Get types

func TestGetTypes (t *testing.T) {
    
    g := ParseString("aa, ab, bb, axx, aj, vv")
    r,_ := g.GetSimilar("a[a-b]")
    
    if r.Len()!=2 || r.Text()!="aa\nab" {
        t.Error("similar")
    }
    
    g = NewGraph("111")
    if i,_ := g.Int64(); i != 111 {
        t.Error("Int64")
    }
    
    g = NewGraph("111.1")
    if i,_ := g.Float64(); i != 111.1 {
        t.Error("Float64")
    }
    
    g = NewGraph(float32(111.2))
    if i,_ := g.Float64(); i != 111.2 {
        t.Error("Float64")
    }
    
    g = NewGraph("true")
    if i,_ := g.Bool(); i != true {
        t.Error("Bool")
    }

    g = ParseString("a 1")
    if i,_ := g.GetInt64("a"); i != 1 {
        t.Error("GetInt64")
    }
    
    g = ParseString("a 1.1")
    if i,_ := g.GetFloat64("a"); i != 1.1 {
        t.Error("GetFloat64")
    }
    
    g = ParseString("a 'false'")
    if i,err := g.GetBool("a"); err!=nil || i != false {
        t.Error("GetBool")
    }
    
    g = ParseString("a 'text'")
    if i,err := g.GetBytes("a"); err!=nil || len(i) != 4 {
        t.Error("GetBytes",len(i))
    }
}

func TestGraphValue(t *testing.T) {

	g := NewGraph(1.0)

	i := g.Value()

	typ := reflect.TypeOf(i).String()

	fmt.Printf("%s\n", typ)

}

func TestGraphNumber(t *testing.T) {

	g := NewGraph([]byte("-1"))

	i := g.Number()

	typ := reflect.TypeOf(i).String()

	fmt.Printf("%s\n", typ)

}

func TestIsInteger(t *testing.T) {
	ss := [...]string{"-1", "2", "9.1", " 14", " - 1", " -1 ", "a", "3a", ""}
	rr := [...]bool{true, true, false, true, false, true, false, false, false}

	for i, s := range ss {
		b := IsInteger(s)
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
	ty := reflect.TypeOf(i).String()

	fmt.Printf("%s\n%v\n", ty, s)

	i = 1.1

	s = _string(i)
	ty = reflect.TypeOf(i).String()

	fmt.Printf("%s\n%v\n", ty, s)
}

// Templates

func TestTemplate1(ts *testing.T) {
	// Context
	g := NewGraph("b")
	g.Add(1)

	t := NewTemplate("a $b")
	t.This = "!t"
	println(t.Text())

	s := t.Process(g)

	println(string(s))
}

func TestTemplateIf(ts *testing.T) {
	// Context
	g := NilGraph()

	t := NewTemplate("$if('false') a $else b $end")
	println(t.Text())

	s := t.Process(g)
	println(string(s))

	t = NewTemplate("$if('true') a $else b $end")
	s = t.Process(g)
	println(string(s))
}

func TestTemplateFor(ts *testing.T) {
	// Context
	g := NilGraph()
	c := g.Add("b")
	c.Add(1)
	c.Add(2)

	t := NewTemplate("$b\n---\n$for(a,b) [$a] $end")

	s := t.Process(g)

	println(string(s))
}

// function.go

func TestFunction1(ts *testing.T) {
	// Context
	g := NilGraph()
	c := g.Add("T")
	ty := c.Add("!type")
	ty.Add("function")
	g.Add("a").Add("title $b")
	g.Add("b").Add("var")

	t := NewTemplate("$T(a)")
	println(t.Text())
	s := t.Process(g)

	println(string(s))
}

type Math struct {
}

func newMath() interface{} {
	return &Math{}
}

func (*Math) Sin(x float64) float64 {
	return math.Sin(x)
}

func TestFunction(t *testing.T) {

	FunctionAddToFactory("math", newMath)

	g := NilGraph()
	f := g.Add("math")
	f.Add("!type").Add("math")

	path := NewPath("math.Sin(1.0)")
	println(path.Text())

	i := g.Eval(path)
	s := reflect.TypeOf(i).String()
	fmt.Printf("%s / %v\n", s, i)
}

// log.go

func TestLog (t *testing.T) {

    file := "/tmp/log.gb"

    log, _ := OpenLog(file)

    g := ParseString("a b, c, d")
    b := g.Binary()
    
    n := log.Add(g)
    m := log.Add(g)
    o := log.AddBinary(b)
    
    if n!=0 || m!=16 || o!=32 {
        t.Error("log.Add",o)
    }
    
    g2,_,n2 := log.Get(0)
    g3,_,n3 := log.Get(n2)
    b2,_,_  := log.GetBinary(n3)
    
    g4 := BinParse(b2)
    
    if ! g.Equal(g2) {
        t.Error("n!=n2")
    }
    
    if ! g.Equal(g3) {
        t.Error("n!=n3")
    }
    
    if ! g.Equal(g4) {
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

	g := ParseString("a b c")
	g.Set("b", "d")

	fmt.Println(g.Text())
	
	// Output:
	// a
	//   b
	//     d
}

func ExampleGraph_Set_a() {

	g := NilGraph()

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
	g := ParseString("a (b 1, c 2, b 3)")
	fmt.Println(g.Get("a.b{0}").Text())
	fmt.Println(g.Get("a.b{1}").Text())
	fmt.Println("---")
	fmt.Println(g.Get("a.b{}").Text())
	// Output:
	// 1
	// 3
	// ---
	// 1
	// 3
}

func ExampleNewTemplate() {
	p := NewTemplate("Hello, $user")

	g := NilGraph()
	g.Add("user").Add("Jenny")

	fmt.Println(string(p.Process(g)))
	// Output:
	// Hello, Jenny
}

func ExampleNewExpression() {
	e := NewExpression("1-2+3")
	g := NilGraph()
	i := g.Eval(e)

	fmt.Println(i)
	// Output:
	// 2
}

func ExampleGraph_Check() {

	schema := ParseString("a !int, b !string, c !float, d !bool")
	g := ParseString("a 1, b s, c 1.0, d true")

	b, message := schema.Check(g)
	fmt.Println(b, message)
	// Output:
	// true
}

func ExampleGraph_Eval() {
	g := NilGraph()
	g.Add("a").Add(4)
	g.Add("b").Add("4")
	e := NewExpression("a+3")
	e2 := NewExpression("b+3")
	fmt.Println(g.Eval(e))
	fmt.Println(g.Eval(e2))
	// Output:
	// 7
	// 43
}
