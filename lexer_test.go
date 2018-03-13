package ogdl

import (
	"fmt"
	"strings"
	"testing"
)

func TestByte(t *testing.T) {
	r := strings.NewReader("hola")

	p := NewParser(r)

	for {
		c, err := p.Byte()
		fmt.Printf("%c %v\n", c, err)
		if err != nil {
			break
		}
	}

}

func TestUnreadByte(t *testing.T) {
	r := strings.NewReader("h")

	p := NewParser(r)

	// read letter
	c, err := p.Byte()
	fmt.Printf("%c %v\n", c, err)

	// read EOS
	c, err = p.Byte()
	fmt.Printf("%c %v\n", c, err)

	// unread EOS
	p.UnreadByte()

	// read EOS
	c, err = p.Byte()
	fmt.Printf("%c %v\n", c, err)

	// 2 x unread: read letter
	p.UnreadByte()
	p.UnreadByte()

	// read letter
	c, err = p.Byte()
	fmt.Printf("%c %v\n", c, err)

}

func TestComment1(t *testing.T) {
	r := strings.NewReader("hola")

	p := NewParser(r)

	p.Comment()

	for {
		c, err := p.Byte()
		fmt.Printf("%c %v\n", c, err)
		if err != nil {
			break
		}
	}

}

func TestEnd(t *testing.T) {
	r := strings.NewReader("hola")

	p := NewParser(r)

	p.End()

	for {
		c, err := p.Byte()
		fmt.Printf("%c %v\n", c, err)
		if err != nil {
			break
		}
	}

	fmt.Printf("end2? %v\n", p.End())

}

func TestBreak(t *testing.T) {
	r := strings.NewReader("hola\nmundo")

	p := NewParser(r)

	s, _ := p.String()
	fmt.Printf("[%s]\n", s)

	c := p.PeekByte()
	fmt.Printf("%c\n", c)

	fmt.Printf("break? %v\n", p.Break())

}

func TestPeekByte(t *testing.T) {
	r := strings.NewReader("hola")

	p := NewParser(r)

	c := p.PeekByte()
	fmt.Printf("%c\n", c)

	for {
		c, err := p.Byte()
		fmt.Printf("%c %v\n", c, err)
		if err != nil {
			break
		}
	}

	p.UnreadByte()
	p.UnreadByte()

	for {
		c, err := p.Byte()
		fmt.Printf("%c %v\n", c, err)
		if err != nil {
			break
		}
	}
}

func TestString(t *testing.T) {
	r := strings.NewReader("hola")

	p := NewParser(r)

	s, b := p.String()

	fmt.Println("string", s, b)

	fmt.Printf("end? %v\n", p.End())
}

func TestScalar(t *testing.T) {
	r := strings.NewReader("hola")

	p := NewParser(r)

	s, b := p.Scalar(0)

	fmt.Println("string", s, b)

	fmt.Printf("end? %v\n", p.End())
}

func TestBlockLex(t *testing.T) {

	r := strings.NewReader("\\\n  hola\n  ")
	p := NewParser(r)

	s, b := p.Block(0)

	if s != "hola" || b != true {
		t.Error()
	}
}

func TestQuoted(t *testing.T) {
	r := strings.NewReader("'hola'")

	p := NewParser(r)

	s, b, _ := p.Quoted(0)

	fmt.Println("string", s, b)

	fmt.Printf("end? %v\n", p.End())
}

func TestProd1(t *testing.T) {

	r := strings.NewReader("\t\t\nhola\n")

	p := NewParser(r)

	i, n := p.Space()
	fmt.Printf("%d %d\n", i, n)

	b := p.Break()
	fmt.Println(b)

}

func TestToken1(t *testing.T) {
	r := strings.NewReader("a")

	p := NewParser(r)

	p.Byte()
	p.UnreadByte()

	s, b := p.Token8()
	fmt.Println(s, b)

	s, b = p.Token8()
	fmt.Println(s, b)
}
