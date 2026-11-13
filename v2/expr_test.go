package ogdl

import (
	"errors"
	"testing"
)

func TestExprParse(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1", "1"},
		{"1.5", "1.5"},
		{"'abc'", "'abc'"},
		{"true", "true"},
		{"a", "token(a)"},
		{"a.b", "token(a) token(b)"},
		{"1+2", "(1+2)"},
		{"1+2*3", "(1+(2*3))"},
		{"(1+2)*3", "((1+2)*3)"},
		{"1-2+3", "((1-2)+3)"}, // left-associative
		{"2*3/4", "((2*3)/4)"},
		{"a==1 && b==2", "((token(a)==1)&&(token(b)==2))"},
		{"a || b && c", "(token(a)||(token(b)&&token(c)))"},
		{"!a", "(!token(a))"},
		{"-1", "(-1)"},
		{"!(a==b)", "(!(token(a)==token(b)))"},
		{"a!=b", "(token(a)!=token(b))"},
		{"f()", "token(f) call()"},
		{"f(1,2)", "token(f) call(1,2)"},
		{"f(1+2)", "token(f) call((1+2))"},
		{"a.b(c)", "token(a) token(b) call(token(c))"},
		{"a[0]", "token(a) index(0)"},
		{"a[i+1]", "token(a) index((token(i)+1))"},
		{"a.(b)", "token(a) group(token(b))"},
		{"a{1}", "token(a) select(1)"},
		{"'true'", "'true'"},
		{"1 + 2", "(1+2)"}, // spaces are not significant
		// Assignment binds loosest and is right-associative (Expr RFC 4.1).
		{"a=1", "(token(a)=1)"},
		{"a = 1 + 2", "(token(a)=(1+2))"},
		{"a = b = 1", "(token(a)=(token(b)=1))"},
		{"a += 1", "(token(a)+=1)"},
		{"a.b *= 2", "(token(a) token(b)*=2)"},
		{"a = b == c", "(token(a)=(token(b)==token(c)))"},
	}

	for _, c := range cases {
		e, err := ParseExpr(c.in)
		if err != nil {
			t.Errorf("ParseExpr(%q): %v", c.in, err)
			continue
		}
		if got := exprString(e); got != c.want {
			t.Errorf("ParseExpr(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestExprParseErrors(t *testing.T) {
	cases := []struct {
		in   string
		want error
	}{
		{"", ErrMissingOperand},
		{"1+", ErrMissingOperand},
		{"a=", ErrMissingOperand},
		{"(1+2", ErrMissingParen},
		{"a&b", ErrReservedOp},
		{"a|b", ErrReservedOp},
		{"'abc", ErrUnterminatedStr},
		{"f(1,)", ErrInvalidPath},
		{"1 2", ErrInvalidExpr},
		{"@", ErrInvalidExpr},
		{"99999999999999999999", ErrNumberRange},
	}

	for _, c := range cases {
		_, err := ParseExpr(c.in)
		if err == nil {
			t.Errorf("ParseExpr(%q): expected %v, got none", c.in, c.want)
			continue
		}
		if !errors.Is(err, c.want) {
			t.Errorf("ParseExpr(%q): expected %v, got %v", c.in, c.want, err)
		}
	}
}

// Strict paths must not acquire expression syntax (Path RFC 5.1).
func TestStrictPathRejectsExtensions(t *testing.T) {
	for _, p := range []string{"a(b)", "a.(b)", "a[i]", "f()"} {
		if _, err := Path(p); err == nil {
			t.Errorf("Path(%q) should be rejected in a strict path", p)
		}
	}
}
