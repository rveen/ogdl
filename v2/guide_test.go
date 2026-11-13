package ogdl

import (
	"errors"
	"strings"
	"testing"
)

// The examples of doc/guide.md. When the guide changes, change these too.

const guideFirewall = `# Firewall for the office network
firewall
  default deny
  # first match wins
  rule allow
    from 10.0.0.0/8
    port 22
  rule allow
    from 192.168.0.0/16
  rule deny
    from any
`

const guidePipeline = `pipeline
  step checkout
    repo git@example.com:app.git
  step run
    cmd "go vet ./..."
  step run
    cmd "go test ./..."
  step publish
    when tag
`

const guideHosts = `hosts
  web
    ip 10.0.0.1
    role frontend
  db
    ip 10.0.0.2
    role storage
    backup daily
`

func guideTree(t *testing.T, s string) *Node {
	t.Helper()
	g, err := ParseString(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return g
}

func sameTree(t *testing.T, a, b string) {
	t.Helper()
	if x, y := guideTree(t, a).Text(), guideTree(t, b).Text(); x != y {
		t.Errorf("trees differ:\n%s\nvs\n%s", x, y)
	}
}

func TestGuideLanguage(t *testing.T) {
	sameTree(t, "rule allow any\n", "rule\n  allow\n  any\n")
	sameTree(t, "rule allow any\n  log yes\n", "rule\n  allow\n  any\n  log\n    yes\n")
	sameTree(t, "S1\n  value 1.0\n  class asil\n  title \"S1 – Slight injury\"\n",
		"S1\n  value\n    1.0\n  class\n    asil\n  title\n    \"S1 – Slight injury\"\n")
	if got := guideTree(t, "S1 value 1.0 class asil\n").N(0).Out; len(got) != 4 {
		t.Errorf("S1 value 1.0 class asil: %d children, want 4", len(got))
	}

	words := guideTree(t, "when S>=S2\ncheck a==b\nargs --verbose --level=3\nurl https://example.com/?q=ogdl\n")
	for path, want := range map[string]string{"when": "S>=S2", "check": "a==b", "url": "https://example.com/?q=ogdl"} {
		if got := words.S(path).String(); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	if got := words.S("args").N(1).ThisString(); got != "--level=3" {
		t.Errorf("--level=3 was split: %q", got)
	}
	if got := guideTree(t, "a k=v\n").N(0).Out; len(got) != 1 || got[0].ThisString() != "k=v" || len(got[0].Out) != 0 {
		t.Errorf("a k=v: want one leaf child k=v, got %s", guideTree(t, "a k=v\n").Text())
	}

	q := guideTree(t, "title \"Hello, world\"\ngreeting 'She said \"hi\"'\npath C:\\Users\\ana\n")
	if q.S("greeting").String() != `She said "hi"` || q.S("path").String() != `C:\Users\ana` {
		t.Errorf("quoting: %s", q.Text())
	}

	ml := guideTree(t, "motto \"two\n       lines\"\nnote \\\n  Line one\n    indented\n  Line three\n")
	if ml.S("motto").String() != "two\nlines" || ml.S("note").String() != "Line one\n  indented\nLine three" {
		t.Errorf("multi-line: %s", ml.Text())
	}

	cm := guideTree(t, "# a comment\nport 8080   # a trailing comment\ncolor #ff0000\n")
	if cm.Text() != "port\n  8080\ncolor\n  #ff0000\n" {
		t.Errorf("comments: %q", cm.Text())
	}

	for _, bad := range []string{"a\n  b\n\tc\n", "a\n    b\n  c\n", "a \"open\n", "a \"\\q\"\n"} {
		var se *SyntaxError
		if _, err := ParseString(bad); !errors.As(err, &se) || se.Line == 0 {
			t.Errorf("%q: expected a located syntax error, got %v", bad, err)
		}
	}
}

func TestGuideExamplesParse(t *testing.T) {
	for _, s := range []string{
		"network\n  eth0\n    ip 192.168.0.10\n  eth1\n    ip 192.168.1.10\n",
		"severity\n  S0 \"No injuries\"\n  S1 \"Light injuries\"\n  S2 \"Severe injuries\"\n  S3 \"Fatal injuries\"\n",
		guidePipeline,
		guideHosts,
		"menu\n  File\n    New\n    Open\n    Recent\n      report.ogdl\n  Edit\n    Undo\n",
	} {
		guideTree(t, s)
	}
	if got := guideTree(t, "severity\n  S0 \"No injuries\"\n").Get("severity.S0").String(); got != "No injuries" {
		t.Errorf("S0 = %q", got)
	}
	p := guideTree(t, guidePipeline)
	if got := p.Get("pipeline.step.repo").String(); got != "git@example.com:app.git" {
		t.Errorf("repo = %q", got)
	}
	if got := p.Get("pipeline.step{1}.cmd").String(); got != "go vet ./..." {
		t.Errorf("cmd = %q", got)
	}
	h := guideTree(t, guideHosts)
	if got := h.Get("hosts.db.backup").String(); got != "daily" {
		t.Errorf("backup = %q", got)
	}
	if got := h.Get("hosts.db.role").String(); got != "storage" {
		t.Errorf("role = %q", got)
	}
}

func TestGuideGo(t *testing.T) {
	g := guideTree(t, guideFirewall)
	if g.Get("firewall.default").String() != "deny" ||
		g.Get("firewall.default").ThisString() != "default" ||
		g.Get("firewall.rule.port").Int64() != 22 ||
		g.Get("firewall.rule{1}.from").String() != "192.168.0.0/16" ||
		g.Get("firewall.missing").String() != "" {
		t.Error("reading values")
	}

	compact, _ := g.Compact()
	if want := "firewall\n  default deny\n  rule allow\n    from 10.0.0.0/8\n    port 22\n  rule allow\n    from 192.168.0.0/16\n  rule deny\n    from any\n"; compact != want {
		t.Errorf("compact:\n%s", compact)
	}

	d, _ := ParseDocumentString(guideFirewall)
	d.Root.Set("firewall.default", "allow")
	fw := d.Root.S("firewall")
	fw.Out = append(fw.Out, &Node{Data: "rule", Out: []*Node{
		{Data: "log"}, {Data: "from", Out: []*Node{{Data: "any"}}},
	}})
	out, _ := d.Format()
	want := strings.Replace(guideFirewall, "default deny", "default allow", 1) + "  rule log\n    from any\n"
	if out != want {
		t.Errorf("document:\n%s", out)
	}
}

func TestGuidePathsAndExpressions(t *testing.T) {
	g := guideTree(t, guideFirewall)
	if g.Get("firewall.rule[0]").ThisString() != "allow" || len(g.Get("firewall.rule{}").Out) != 3 {
		t.Error("paths")
	}
	h := guideTree(t, "hosts\n  web-1\n    ip 1\n")
	if h.Get("hosts.'web-1'.ip").String() != "1" {
		t.Error("quoted path")
	}
	for expr, want := range map[string]interface{}{
		"firewall.rule.port + 1":     int64(23),
		"firewall.default == 'deny'": true,
		"firewall.rule{2}[0]":        "deny",
		"'4' + 3":                    int64(7),
	} {
		v, err := g.EvalString(expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if n, ok := v.(*Node); ok {
			v = n.ThisString()
		}
		if v != want {
			t.Errorf("%s = %#v, want %#v", expr, v, want)
		}
	}
}

const guideSchema = `!type action allow deny

firewall
  default !action
  rule+ !action
    from !string
    port? !int
`

func TestGuideSchema(t *testing.T) {
	s, err := ParseSchema(guideSchema)
	if err != nil {
		t.Fatal(err)
	}
	g := guideTree(t, guideFirewall)
	if err := s.Validate(g); err != nil {
		t.Fatal(err)
	}

	bad := guideTree(t, "firewall\n  default drop\n  rule allow\n    from any\n    port ssh\n  rule alow\n    form x\n")
	var ve *ValidationError
	if !errors.As(s.Validate(bad), &ve) {
		t.Fatal("expected a validation error")
	}
	var got []string
	for _, is := range ve.Issues {
		got = append(got, is.String())
	}
	want := []string{
		`firewall.default[0]: "drop" is not !action ("allow", "deny")`,
		`firewall.rule.port[0]: "ssh" is not !int`,
		`firewall.rule{1}: missing from`,
		`firewall.rule{1}[0]: "alow" is not !action ("allow", "deny")`,
		`firewall.rule{1}.form: unexpected; expected one of from, port`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("issues:\n%s", strings.Join(got, "\n"))
	}

	if err := s.Apply(g); err != nil {
		t.Fatal(err)
	}
	if port, ok := g.Get("firewall.rule.port[0]").Data.(int64); !ok || port != 22 {
		t.Errorf("Apply: port = %#v", g.Get("firewall.rule.port[0]").Data)
	}

	for _, def := range []string{"!type action allow deny", "!type number !int !float", "!type limit !int unlimited", "!type port !int"} {
		if _, err := ParseSchema(def + "\n"); err != nil {
			t.Errorf("%s: %v", def, err)
		}
	}
	if _, err := ParseSchema("port=!int\n"); err == nil || !strings.Contains(err.Error(), "port !int") {
		t.Errorf("port=!int: %v", err)
	}
	maps, _ := ParseSchema("hosts\n  !string*\n    ip !string\n    role? !string\n")
	if err := maps.Validate(guideTree(t, "hosts\n  web\n    ip 10.0.0.1\n    role frontend\n  db\n    ip 10.0.0.2\n")); err != nil {
		t.Error(err)
	}
	args, _ := ParseSchema("args !int* !string\n")
	if args.Validate(guideTree(t, "args 1 2 x\n")) != nil || args.Validate(guideTree(t, "args 1 2 3\n")) == nil {
		t.Error("no backtracking example")
	}
}

func guideRender(t *testing.T, tpl string, ctx *Node, html bool) string {
	t.Helper()
	var out string
	var err error
	if html {
		out, err = ProcessHTMLString(tpl, ctx)
	} else {
		out, err = ProcessString(tpl, ctx)
	}
	if err != nil {
		t.Fatalf("template %q: %v", tpl, err)
	}
	return out
}

func TestGuideTemplates(t *testing.T) {
	g := guideTree(t, guideFirewall)
	cases := []struct{ tpl, want string }{
		{"Default policy: $firewall.default", "Default policy: deny"},
		{"$if(firewall.default == 'deny')Closed by default.$else Open by default.$end", "Closed by default."},
		{"$for(r, firewall.rule{})$r[0] $r.from$if(r.port) port $r.port$end\n$end",
			"allow 10.0.0.0/8 port 22\nallow 192.168.0.0/16\ndeny any\n"},
		{"$for(r, firewall.rule{})$if(r[0] == 'deny')$break$end$r.from\n$end",
			"10.0.0.0/8\n192.168.0.0/16\n"},
		{"$(n = 0)$for(r, firewall.rule{})$(n += 1)${n}. $r[0] $r.from\n$end",
			"1. allow 10.0.0.0/8\n2. allow 192.168.0.0/16\n3. deny any\n"},
		{"Price: $5.00, or $$x", "Price: $5.00, or $x"},
	}
	for _, c := range cases {
		if got := guideRender(t, c.tpl, guideTree(t, guideFirewall), false); got != c.want {
			t.Errorf("%q:\ngot  %q\nwant %q", c.tpl, got, c.want)
		}
	}

	// The trap of section 8.1.
	if _, err := ProcessString("$(n = 1)$n. done", g); err == nil {
		t.Error("$n. should be an error")
	}

	team := guideTree(t, "team\n  member\n    name Ana\n    role lead\n  member\n    name Bo\n    role dev\n  member\n    name \"<Eve>\"\n    role dev\n")
	got := guideRender(t, "<ul>\n$for(m, team.member{})  <li>$m.name$if(m.role == 'lead') (lead)$end</li>\n$end</ul>", team, true)
	if want := "<ul>\n  <li>Ana (lead)</li>\n  <li>Bo</li>\n  <li>&lt;Eve&gt;</li>\n</ul>"; got != want {
		t.Errorf("html:\n%s", got)
	}

	for _, bad := range []string{"$if(a)x", "$end", "$break", "$(1/0)"} {
		if _, err := ProcessString(bad, g); err == nil {
			t.Errorf("%q: expected an error", bad)
		}
	}
}
