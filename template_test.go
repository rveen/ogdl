package ogdl

import (
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
	g := New()
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
	g := New()
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
	g := New()
	item := g.Add("result").Add("item")
	item.Add("x0").Add("yo")
	item.Add("id").Add("1")

	tpl := NewTemplate("$(a=result.item)$a.x0")

	s := string(tpl.Process(g))

	if s != "yo" {
		t.Error("e1", s)
	}
}
