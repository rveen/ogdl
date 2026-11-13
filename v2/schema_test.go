package ogdl

import (
	"errors"
	"sort"
	"strings"
	"testing"
)

// schemaCase is a document and the issues it must produce, as "path: message
// fragment"; none means valid.
type schemaCase struct {
	doc    string
	issues []string
}

func runSchemaCases(t *testing.T, schema string, cases []schemaCase) {
	t.Helper()
	s, err := ParseSchema(schema)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	for _, c := range cases {
		doc, err := ParseString(c.doc)
		if err != nil {
			t.Fatalf("document %q: %v", c.doc, err)
		}
		err = s.Validate(doc)
		var got []Issue
		if err != nil {
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("document %q: expected *ValidationError, got %T", c.doc, err)
			}
			got = ve.Issues
		}
		if len(got) != len(c.issues) {
			t.Errorf("document %q: got %d issues %v, want %v", c.doc, len(got), got, c.issues)
			continue
		}
		var gs []string
		for _, is := range got {
			gs = append(gs, is.String())
		}
		sort.Strings(gs)
		want := append([]string(nil), c.issues...)
		sort.Strings(want)
		for i := range want {
			path, frag, _ := strings.Cut(want[i], ": ")
			if !strings.HasPrefix(gs[i], path+": ") || !strings.Contains(gs[i], frag) {
				t.Errorf("document %q: issue %q, want %q", c.doc, gs[i], want[i])
			}
		}
	}
}

// Appendix A.1
func TestSchemaValues(t *testing.T) {
	runSchemaCases(t, "server\n  host !string\n  port !int\n  timeout? !float\n  debug? !bool\n", []schemaCase{
		{"server\n  host a\n  port 80\n", nil},
		{"server\n  port 80\n  host a\n  timeout 1.5\n  debug true\n", nil},
		{"server\n  host a\n  port 80\n  timeout 2\n", nil},
		{"server\n  host a\n", []string{"server: missing port"}},
		{"server\n  host a\n  port 80\n  port 81\n", []string{"server: too many port"}},
		{"server\n  host a\n  port 80.0\n", []string{`server.port[0]: "80.0" is not !int`}},
		{"server\n  host a\n  port 80\n  debug yes\n", []string{`server.debug[0]: "yes" is not !bool`}},
		{"server\n  host a\n  port 99999999999999999999\n", []string{"server.port[0]: is not !int"}},
		{"server\n  host a\n  port 80\n  extra 1\n", []string{"server.extra: unexpected"}},
		{"server\n  host \"\"\n  port -1\n", nil},
		{"server host=a port=80\n", []string{"server: missing host", "server: missing port", "server.'host=a': unexpected", "server.'port=80': unexpected"}},
		{"", []string{"(root): missing server"}},
	})
}

// Appendix A.2
func TestSchemaArguments(t *testing.T) {
	runSchemaCases(t, "!type action allow deny\nrule* !action !string\n  log? !bool\n", []schemaCase{
		{"", nil},
		{"rule allow any\n", nil},
		{"rule deny 10.0.0.0/8\n  log true\n", nil},
		{"rule allow\n", []string{"rule: missing !string"}},
		{"rule permit any\n", []string{`rule[0]: "permit" is not !action ("allow", "deny")`}},
		{"rule allow any extra\n", []string{"rule.extra: unexpected"}},
		{"rule allow any log true\n", []string{"rule.log: missing !bool", "rule.true: unexpected"}},
		{"rule allow any\nrule deny any\nrule maybe x\n", []string{`rule{2}[0]: "maybe" is not !action`}},
	})
}

// Appendix A.3
func TestSchemaMaps(t *testing.T) {
	runSchemaCases(t, "hosts\n  !string*\n    ip !string\n", []schemaCase{
		{"hosts\n", nil},
		{"hosts\n  web\n    ip 10.0.0.1\n  db\n    ip 10.0.0.2\n", nil},
		{"hosts\n  web\n", []string{"hosts.web: missing ip"}},
	})
}

// Appendix A.4
func TestSchemaNoBacktracking(t *testing.T) {
	runSchemaCases(t, "args !int* !string\n", []schemaCase{
		{"args 1 2 x\n", nil},
		{"args 1 2 3\n", []string{"args: missing !string"}},
	})
}

// Appendix A.5
func TestSchemaLeaves(t *testing.T) {
	runSchemaCases(t, "enabled?\nport !int\nmeta? !any*\n", []schemaCase{
		{"enabled\nport 1\n", nil},
		{"enabled yes\nport 1\n", []string{"enabled.yes: unexpected"}},
		{"port\n  1 x\n", []string{"port.1.x: unexpected"}},
		{"port 1\nmeta\n", nil},
		{"port 1\n  x 2\n", []string{"port.x: unexpected"}},
		{"port 1\nmeta a b\n  c d\n    e\n", nil},
	})
}

// Appendix A.6
func TestSchemaEscapes(t *testing.T) {
	runSchemaCases(t, "\\!important? !bool\nsize\\* !int\n", []schemaCase{
		{"!important true\nsize* 3\n", nil},
		{"size 3\n", []string{"(root): missing size*", "size: unexpected"}},
	})
}

// Appendix A.7
func TestSchemaErrors(t *testing.T) {
	for _, s := range []string{
		"port?=!int\n",
		"port=!int\n",
		"a !\n",
		"a !integer\n",
		"a !type\n",
		"!type int !string\n",
		"!type a !b\n!type b !a\n",
		"!type* a x\n",
		"!type a* x\n",
		"!type a\n",
		"!type\n",
		"a !int\na !string\n",
		"a\n  !type t x\n",
		"a\\\n",
		"!type a x\n!type a y\n",
		"!type a x\n  y z\n",
	} {
		_, err := ParseSchema(s)
		if !errors.Is(err, ErrSchema) {
			t.Errorf("schema %q: expected ErrSchema, got %v", s, err)
		}
	}
}

func TestSchemaErrorHint(t *testing.T) {
	for in, hint := range map[string]string{"port?=!int\n": `"port? !int"`, "port=!int\n": `"port !int"`} {
		_, err := ParseSchema(in)
		if err == nil || !strings.Contains(err.Error(), hint) {
			t.Errorf("schema %q: expected a hint to write %s, got %v", in, hint, err)
		}
	}
}

// Appendix A.8
func TestSchemaRootMapAndTyping(t *testing.T) {
	schema := "!type number !int !float\n!string*\n  n !number\n  on !bool\n"
	runSchemaCases(t, schema, []schemaCase{
		{"a\n  n 1\n  on true\nb\n  n 2.5\n  on false\n", nil},
		{"a\n  n 0x10\n  on true\n", []string{`a.n[0]: "0x10" is not !number`}},
	})

	s, _ := ParseSchema(schema)
	doc, _ := ParseString("a\n  n 1\n  on true\nb\n  n 2.5\n  on false\n")
	if err := s.Apply(doc); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		path string
		want interface{}
	}{
		{"a.n[0]", int64(1)},
		{"b.n[0]", 2.5},
		{"a.on[0]", true},
		{"b.on[0]", false},
	}
	for _, c := range checks {
		if got := doc.Get(c.path).Data; got != c.want {
			t.Errorf("%s = %#v (%T), want %#v", c.path, got, got, c.want)
		}
	}
	// Typed values still serialize as their text.
	if got, want := doc.Text(), "a\n  n\n    1\n  on\n    true\nb\n  n\n    2.5\n  on\n    false\n"; got != want {
		t.Errorf("text after Apply: %q, want %q", got, want)
	}
}

func TestSchemaApplyInvalidLeavesDocument(t *testing.T) {
	s, _ := ParseSchema("a !int\nb !int\n")
	doc, _ := ParseString("a 1\nb x\n")
	if err := s.Apply(doc); err == nil {
		t.Fatal("expected a validation error")
	}
	if _, ok := doc.Get("a[0]").Data.(string); !ok {
		t.Fatal("Apply changed an invalid document")
	}
}

func TestSchemaEnumLiteralNotConverted(t *testing.T) {
	s, _ := ParseSchema("!type limit !int unlimited\nmax !limit\n")
	for _, c := range []struct {
		doc  string
		want interface{}
	}{{"max 5\n", int64(5)}, {"max unlimited\n", "unlimited"}} {
		doc, _ := ParseString(c.doc)
		if err := s.Apply(doc); err != nil {
			t.Fatal(err)
		}
		if got := doc.Get("max[0]").Data; got != c.want {
			t.Errorf("%q: got %#v, want %#v", c.doc, got, c.want)
		}
	}
}

// The example of the RFC's introduction and section 6.
func TestSchemaIntroduction(t *testing.T) {
	schema := "!type action allow deny\n\nserver\n  host !string\n  port !int\n  timeout? !float\n  debug? !bool\n  rule* !action !string\n"
	runSchemaCases(t, schema, []schemaCase{
		{"server\n  host db.example.com\n  port 8080\n  debug true\n  rule allow 10.0.0.0/8\n  rule deny any\n", nil},
		{"server\n  host db.example.com\n  port 80x\n  prot 8080\n  rule allow 10.0.0.0/8\n  rule maybe any\n", []string{
			`server.port[0]: "80x" is not !int`,
			"server.prot: unexpected; expected one of host, port, timeout, debug, rule",
			`server.rule{1}[0]: "maybe" is not !action ("allow", "deny")`,
		}},
	})
}

func TestDecimalGrammar(t *testing.T) {
	for _, s := range []string{"0", "-1", "+1", "1.", ".5", "1.5e3", "1E-3", "-.5e+2"} {
		if !isDecimal(s) {
			t.Errorf("%q should be decimal", s)
		}
	}
	for _, s := range []string{"", ".", "-", "e3", "1e", "0x10", "0x1p4", "Inf", "NaN", "1_000", "1.2.3", " 1", "1 "} {
		if isDecimal(s) {
			t.Errorf("%q should not be decimal", s)
		}
	}
}

func TestPathElementQuoting(t *testing.T) {
	cases := map[string]string{"web": "web", "eth0-1": "'eth0-1'", "it's": `'it\'s'`, "": "''", `a\b`: `'a\\b'`}
	for in, want := range cases {
		if got := pathElement(in); got != want {
			t.Errorf("pathElement(%q) = %q, want %q", in, got, want)
		}
		if in != "" {
			// The rendered element addresses the node.
			root := &Node{Out: []*Node{{Data: in, Out: []*Node{{Data: "v"}}}}}
			if got := root.Get(pathElement(in)).String(); got != "v" {
				t.Errorf("Get(%q) did not find %q", pathElement(in), in)
			}
		}
	}
}
