package ogdl

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ErrSchema is wrapped by every *SchemaError: a schema that is not
// well-formed (Schema RFC 8).
var ErrSchema = errors.New("ill-formed schema")

// SchemaError reports an ill-formed schema. Name is the schema node at fault,
// as written.
type SchemaError struct {
	Name string
	Msg  string
}

func (e *SchemaError) Error() string {
	return fmt.Sprintf("ogdl schema: %q: %s", e.Name, e.Msg)
}

func (e *SchemaError) Unwrap() error { return ErrSchema }

func schemaErr(name, format string, args ...interface{}) error {
	return &SchemaError{Name: name, Msg: fmt.Sprintf(format, args...)}
}

// Issue is one way in which a document fails its schema (Schema RFC 6). Path
// locates the node in path syntax; it is empty for the root.
type Issue struct {
	Path    string
	Message string
}

func (i Issue) String() string {
	p := i.Path
	if p == "" {
		p = "(root)"
	}
	return p + ": " + i.Message
}

// ValidationError lists every issue found in a document.
type ValidationError struct {
	Issues []Issue
}

func (e *ValidationError) Error() string {
	s := make([]string, len(e.Issues))
	for i, is := range e.Issues {
		s[i] = is.String()
	}
	return "ogdl: document does not match schema: " + strings.Join(s, "; ")
}

// Schema is a compiled schema (Schema RFC). It is immutable and may be used
// concurrently.
type Schema struct {
	root  []*particle
	types map[string]*stype
}

// typeKind distinguishes the built-in types from defined ones.
type typeKind int

const (
	kindDefined typeKind = iota
	kindString
	kindInt
	kindFloat
	kindBool
	kindAny
)

// stype is a type: a predicate on a node's name (Schema RFC 4).
type stype struct {
	name string
	kind typeKind
	alts []alt // defined types only
}

// alt is one alternative of a defined type: a literal name, or a type.
type alt struct {
	lit  string
	ref  string // the referenced type's name, before resolution
	typ  *stype
	node string // as written, for errors
}

// particle describes the document nodes it matches (Schema RFC 2).
type particle struct {
	name     string // named particle: the literal name
	typ      *stype // type particle: the type
	min, max int    // occurrence; max < 0 is unbounded
	children []*particle
	label    string // name or !type, for messages
}

// ParseSchema parses and compiles a schema written as an OGDL document.
func ParseSchema(s string) (*Schema, error) {
	n, err := ParseString(s)
	if err != nil {
		return nil, err
	}
	return CompileSchema(n)
}

// CompileSchema compiles a schema from an already parsed document.
func CompileSchema(root *Node) (*Schema, error) {
	s := &Schema{types: map[string]*stype{
		"string": {name: "string", kind: kindString},
		"int":    {name: "int", kind: kindInt},
		"float":  {name: "float", kind: kindFloat},
		"bool":   {name: "bool", kind: kindBool},
		"any":    {name: "any", kind: kindAny},
	}}

	// Definitions first, so that references may precede them (RFC 4.2).
	var defined []*stype
	var rest []*Node
	for _, c := range root.Out {
		raw := nameString(c.Data)
		sn, err := decodeName(raw)
		if err != nil {
			return nil, err
		}
		if sn.directive && sn.text == "type" {
			if sn.occ {
				return nil, schemaErr(raw, "a type definition takes no occurrence mark")
			}
			t, err := s.define(c)
			if err != nil {
				return nil, err
			}
			defined = append(defined, t)
			continue
		}
		rest = append(rest, c)
	}
	for _, t := range defined {
		for i := range t.alts {
			a := &t.alts[i]
			if a.ref == "" {
				continue
			}
			if a.typ = s.types[a.ref]; a.typ == nil {
				return nil, schemaErr(a.node, "undefined type !%s", a.ref)
			}
		}
	}
	if err := checkCycles(defined); err != nil {
		return nil, err
	}

	var err error
	s.root, err = s.particles(rest)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// define compiles a type definition, `!type name alternative...`.
func (s *Schema) define(c *Node) (*stype, error) {
	raw := nameString(c.Data)
	if len(c.Out) == 0 {
		return nil, schemaErr(raw, "a type definition needs a name")
	}
	nameNode := c.Out[0]
	nraw := nameString(nameNode.Data)
	sn, err := decodeName(nraw)
	if err != nil {
		return nil, err
	}
	if sn.directive || sn.occ || !isTypeName(sn.text) || len(nameNode.Out) > 0 {
		return nil, schemaErr(nraw, "not a type name")
	}
	if sn.text == "type" || s.types[sn.text] != nil {
		return nil, schemaErr(nraw, "type !%s is already defined", sn.text)
	}
	if len(c.Out) == 1 {
		return nil, schemaErr(nraw, "type !%s has no alternatives", sn.text)
	}
	t := &stype{name: sn.text, kind: kindDefined}
	for _, a := range c.Out[1:] {
		araw := nameString(a.Data)
		an, err := decodeName(araw)
		if err != nil {
			return nil, err
		}
		if an.occ || len(a.Out) > 0 {
			return nil, schemaErr(araw, "an alternative is a single name, without occurrence mark or children")
		}
		if an.directive {
			if an.text == "type" {
				return nil, schemaErr(araw, "!type is not a type")
			}
			t.alts = append(t.alts, alt{ref: an.text, node: araw})
		} else {
			t.alts = append(t.alts, alt{lit: an.text, node: araw})
		}
	}
	s.types[t.name] = t
	return t, nil
}

// checkCycles rejects definitions that refer to themselves, directly or not.
func checkCycles(defined []*stype) error {
	state := map[*stype]int{} // 1 visiting, 2 done
	var visit func(t *stype) error
	visit = func(t *stype) error {
		switch state[t] {
		case 1:
			return schemaErr("!"+t.name, "type definitions form a cycle")
		case 2:
			return nil
		}
		state[t] = 1
		for _, a := range t.alts {
			if a.typ != nil && a.typ.kind == kindDefined {
				if err := visit(a.typ); err != nil {
					return err
				}
			}
		}
		state[t] = 2
		return nil
	}
	for _, t := range defined {
		if err := visit(t); err != nil {
			return err
		}
	}
	return nil
}

// particles compiles the schema nodes describing one node's children.
func (s *Schema) particles(nodes []*Node) ([]*particle, error) {
	var ps []*particle
	seen := map[string]bool{}
	for _, c := range nodes {
		raw := nameString(c.Data)
		sn, err := decodeName(raw)
		if err != nil {
			return nil, err
		}
		p := &particle{min: 1, max: 1}
		if sn.occ {
			p.min, p.max = sn.min, sn.max
		}
		if sn.directive {
			if sn.text == "type" {
				return nil, schemaErr(raw, "a type definition must be at depth zero")
			}
			if p.typ = s.types[sn.text]; p.typ == nil {
				return nil, schemaErr(raw, "undefined type !%s", sn.text)
			}
			p.label = "!" + sn.text
		} else {
			if seen[sn.text] {
				return nil, schemaErr(raw, "%s is described twice", sn.text)
			}
			seen[sn.text] = true
			p.name = sn.text
			p.label = sn.text
		}
		if p.children, err = s.particles(c.Out); err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

// schemaName is a decoded schema name (Schema RFC 3).
type schemaName struct {
	directive bool   // began with an unescaped !
	text      string // the name or type name, unescaped, without ! or mark
	occ       bool   // had an occurrence mark
	min, max  int
}

func decodeName(raw string) (schemaName, error) {
	var sn schemaName
	var out []rune
	var esc []bool
	rs := []rune(raw)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch r {
		case '\\':
			if i == len(rs)-1 {
				return sn, schemaErr(raw, `"\" at the end of a name`)
			}
			i++
			out = append(out, rs[i])
			esc = append(esc, true)
			continue
		case '=':
			// `=` has no meaning in OGDL, so `port=!int` is one word.
			// Rejecting it catches schemas written for pairs, and keeps
			// `=` free for constraints (Schema RFC 3.4, 10).
			msg := `unescaped "=" in a name`
			if i > 0 && i < len(rs)-1 {
				msg += fmt.Sprintf("; write %q", string(rs[:i])+" "+string(rs[i+1:]))
			}
			return sn, schemaErr(raw, "%s", msg)
		}
		out = append(out, r)
		esc = append(esc, false)
	}
	if n := len(out); n > 0 && !esc[n-1] {
		switch out[n-1] {
		case '?':
			sn.occ, sn.min, sn.max = true, 0, 1
		case '*':
			sn.occ, sn.min, sn.max = true, 0, -1
		case '+':
			sn.occ, sn.min, sn.max = true, 1, -1
		}
		if sn.occ {
			out = out[:n-1]
		}
	}
	if len(out) > 0 && !esc[0] && out[0] == '!' {
		sn.directive = true
		sn.text = string(out[1:])
		if !isTypeName(sn.text) {
			return sn, schemaErr(raw, "not a type")
		}
		return sn, nil
	}
	sn.text = string(out)
	return sn, nil
}

// isTypeName reports whether s is a type name (Schema RFC 3.1).
func isTypeName(s string) bool {
	for i, r := range s {
		if i == 0 && !unicode.IsLetter(r) {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	return s != ""
}

// matches reports whether a name is a value of the type (Schema RFC 4).
func (t *stype) matches(s string) bool {
	switch t.kind {
	case kindString, kindAny:
		return true
	case kindInt:
		// Base 10 accepts exactly [+-]DIGIT+, and rejects what overflows.
		_, err := strconv.ParseInt(s, 10, 64)
		return err == nil
	case kindFloat:
		// The grammar excludes hex, Inf and NaN; ParseFloat excludes overflow.
		_, err := strconv.ParseFloat(s, 64)
		return isDecimal(s) && err == nil
	case kindBool:
		return s == "true" || s == "false"
	}
	for _, a := range t.alts {
		if a.typ == nil && a.lit == s || a.typ != nil && a.typ.matches(s) {
			return true
		}
	}
	return false
}

// convert returns the typed value a name denotes under the type, if the type
// converts at all (Schema RFC 7).
func (t *stype) convert(s string) (interface{}, bool) {
	switch t.kind {
	case kindInt:
		i, err := strconv.ParseInt(s, 10, 64)
		return i, err == nil
	case kindFloat:
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	case kindBool:
		return s == "true", true
	case kindDefined:
		for _, a := range t.alts {
			if a.typ == nil {
				if a.lit == s {
					return nil, false
				}
			} else if a.typ.matches(s) {
				return a.typ.convert(s)
			}
		}
	}
	return nil, false
}

// describe renders the type for messages: `!action (allow, deny)`.
func (t *stype) describe() string {
	if t.kind != kindDefined {
		return "!" + t.name
	}
	var as []string
	for _, a := range t.alts {
		if a.typ != nil {
			as = append(as, "!"+a.typ.name)
		} else {
			as = append(as, strconv.Quote(a.lit))
		}
	}
	return "!" + t.name + " (" + strings.Join(as, ", ") + ")"
}

// isDecimal reports whether s is a decimal number, integer or not:
// [+-] (DIGIT+ ["." DIGIT*] | "." DIGIT+) [(e|E) [+-] DIGIT+] (Schema RFC 4.1).
func isDecimal(s string) bool {
	if s != "" && (s[0] == '+' || s[0] == '-') {
		s = s[1:]
	}
	n := digits(s)
	s = s[n:]
	if strings.HasPrefix(s, ".") {
		f := digits(s[1:])
		if n == 0 && f == 0 {
			return false
		}
		s = s[1+f:]
	} else if n == 0 {
		return false
	}
	if s != "" && (s[0] == 'e' || s[0] == 'E') {
		s = s[1:]
		if s != "" && (s[0] == '+' || s[0] == '-') {
			s = s[1:]
		}
		e := digits(s)
		if e == 0 {
			return false
		}
		s = s[e:]
	}
	return s == ""
}

// digits counts the leading ASCII digits of s.
func digits(s string) int {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i
}

// Validate checks a document's root node against the schema and reports every
// issue as a *ValidationError (Schema RFC 5, 6).
func (s *Schema) Validate(doc *Node) error {
	v := validator{}
	v.children(doc, s.root, "")
	if len(v.issues) > 0 {
		return &ValidationError{Issues: v.issues}
	}
	return nil
}

// Apply validates the document and, if it is valid, replaces the data of every
// node matched by !int, !float or !bool — directly or through a defined type —
// with an int64, float64 or bool (Schema RFC 7). An invalid document is left
// unchanged.
func (s *Schema) Apply(doc *Node) error {
	v := validator{}
	v.children(doc, s.root, "")
	if len(v.issues) > 0 {
		return &ValidationError{Issues: v.issues}
	}
	for _, m := range v.typed {
		if x, ok := m.typ.convert(nameString(m.node.Data)); ok {
			m.node.Data = x
		}
	}
	return nil
}

type validator struct {
	issues []Issue
	typed  []typedNode
}

type typedNode struct {
	node *Node
	typ  *stype
}

func (v *validator) issue(path, format string, args ...interface{}) {
	v.issues = append(v.issues, Issue{Path: path, Message: fmt.Sprintf(format, args...)})
}

// children matches the children of n against the particles ps (RFC 5.1).
func (v *validator) children(n *Node, ps []*particle, path string) {
	named := map[string]*particle{}
	var types []*particle
	for _, p := range ps {
		if p.typ != nil {
			types = append(types, p)
		} else {
			named[p.name] = p
		}
	}

	// Step 1: named particles, by name, in any order.
	counts := map[*particle]int{}
	reported := map[*particle]bool{}
	var rest []int
	for k, c := range n.Out {
		name := nameString(c.Data)
		p := named[name]
		if p == nil {
			rest = append(rest, k)
			continue
		}
		counts[p]++
		if p.max >= 0 && counts[p] > p.max {
			if !reported[p] {
				v.issue(path, "too many %s: at most %d", p.label, p.max)
				reported[p] = true
			}
			continue
		}
		v.node(c, p, namePath(path, n, k))
	}
	for _, p := range ps {
		if p.typ == nil && counts[p] < p.min {
			v.missing(path, p)
		}
	}

	// Step 2: type particles, in order, possessively.
	i := 0
	for _, p := range types {
		taken := 0
		for i < len(rest) && (p.max < 0 || taken < p.max) {
			c := n.Out[rest[i]]
			if !p.typ.matches(nameString(c.Data)) {
				break
			}
			v.node(c, p, namePath(path, n, rest[i]))
			i++
			taken++
		}
		if taken < p.min {
			if i < len(rest) {
				// Report the mismatch and let the particle consume the child,
				// so that one wrong value is not also reported as unexpected.
				c := n.Out[rest[i]]
				v.issue(indexPath(path, rest[i]), "%q is not %s", nameString(c.Data), p.typ.describe())
				i++
			} else {
				v.missing(path, p)
			}
		}
	}

	// Step 3: the schema is closed.
	for _, k := range rest[i:] {
		v.issue(namePath(path, n, k), "unexpected%s", expected(ps))
	}
}

// node checks a matched node against its particle's children (RFC 5.3).
func (v *validator) node(c *Node, p *particle, path string) {
	if p.typ != nil {
		v.typed = append(v.typed, typedNode{c, p.typ})
		if p.typ.kind == kindAny {
			return
		}
	}
	v.children(c, p.children, path)
}

func (v *validator) missing(path string, p *particle) {
	if p.min > 1 || p.max < 0 && p.min == 1 {
		v.issue(path, "missing %s: at least %d", p.label, p.min)
		return
	}
	v.issue(path, "missing %s", p.label)
}

// expected lists the named particles, for an unexpected child.
func expected(ps []*particle) string {
	var names []string
	for _, p := range ps {
		if p.typ == nil {
			names = append(names, p.name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "; expected one of " + strings.Join(names, ", ")
}

// namePath extends path with the k-th child of n, addressed by name, with an
// occurrence selector if earlier siblings share the name (Path RFC 6).
func namePath(path string, n *Node, k int) string {
	name := nameString(n.Out[k].Data)
	occ := 0
	for _, c := range n.Out[:k] {
		if nameString(c.Data) == name {
			occ++
		}
	}
	e := pathElement(name)
	if occ > 0 {
		e += "{" + strconv.Itoa(occ) + "}"
	}
	if path == "" {
		return e
	}
	return path + "." + e
}

// indexPath extends path with an index.
func indexPath(path string, k int) string {
	return path + "[" + strconv.Itoa(k) + "]"
}

// pathElement writes a name as a path element: a token if it is one, quoted
// otherwise (Path RFC 4).
func pathElement(name string) string {
	token := name != ""
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			token = false
			break
		}
	}
	if token {
		return name
	}
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + r.Replace(name) + "'"
}
