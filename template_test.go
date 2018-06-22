package ogdl

import (
	"fmt"
	"testing"
)

func template(context *Graph, tpl string) []byte {
	t := NewTemplate(tpl)
	return t.Process(context)
}

func hello() string {
	return "hello"
}

func TestTemplate10(t *testing.T) {
	// Context
	g := New("_")
	g.Add("b").Add(1)
	g.Set("T", template)
	g.Set("H", hello)
	g.Set("header", "<html>")

	tpl := NewTemplate("$T(_this,header)")

	s := string(tpl.Process(g))
	if s != "<html>" {
		t.Error("e1", s)
	}

	tpl = NewTemplate("$H()")

	s = string(tpl.Process(g))
	if s != "hello" {
		t.Error("e2", s)
	}
}

func TestTemplate11(t *testing.T) {
	// Context
	g := New(nil)
	item := g.Add("result").Add("item")
	item.Add("x0").Add("yo")
	item.Add("id").Add("1")

	tpl := NewTemplate("$result[0]")

	s := string(tpl.Process(g))

	if s != "item\n  x0\n    yo\n  id\n    1" {
		t.Error("e1", s)
	}
}

func TestTemplate12(t *testing.T) {
	// Context
	g := New(nil)
	item := g.Add("result").Add("item")
	item.Add("x0").Add("yo")
	item.Add("id").Add("1")

	tpl := NewTemplate("$(a=result.item)$a.x0")

	s := string(tpl.Process(g))

	if s != "yo" {
		t.Error("e1", s)
	}
}

// Templates

func TestTemplate1(ts *testing.T) {
	// Context
	g := New(nil)
	g.Add("b").Add(1)

	t := NewTemplate("a $b")

	s := t.Process(g)

	if string(s) != "a 1" {
		ts.Error("template", string(s))
	}
}

func TestTemplateOperatorConfusion(ts *testing.T) {
	// Context
	g := New(nil)
	g.Add("b").Add(1)

	t := NewTemplate("$(a='/') $a $b")

	s := t.Process(g)

	if string(s) != " / 1" {
		ts.Error("template", string(s))
	}
}

func TestTemplateIfEmptyString(ts *testing.T) {
	// Context
	g := New(nil)
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
	g := New(nil)

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

func TestTemplateFor1(ts *testing.T) {
	// Context
	g := New(nil)
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

func ExampleGraph_Process_for() {

	g := FromString("spaces\n  cvm\n  req\n    stkreq\n    sysreq\n  design\n    hardware")
	t := NewTemplate("$spaces\n$for(s,spaces)$s._string\n$for(d,s[0])- $d\n$end$end")
	s := t.Process(g)

	fmt.Println(string(s))
	// Output:
	// cvm
	// req
	//   stkreq
	//   sysreq
	// design
	//   hardware
	// cvm
	// req
	// - stkreq
	// - sysreq
	// design
	// - hardware
}

func ExampleNewTemplate() {
	p := NewTemplate("Hello, $user")

	g := New("_")
	g.Add("user").Add("Jenny")

	fmt.Println(string(p.Process(g)))
	// Output:
	// Hello, Jenny
}
