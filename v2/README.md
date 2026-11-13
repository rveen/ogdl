# ogdl

A Go implementation of **OGDL**, the Ordered Graph Data Language, together with
the path, expression and template languages built on it.

OGDL is a minimal, indentation-based text format for **ordered trees**. Unlike
JSON, YAML or TOML, which ask you to choose between maps (named, unordered) and
lists (ordered, unnamed), OGDL has one structure: at every level, an ordered
list of named nodes. Names may repeat and order is always kept, so a document
can depend on declaration order (rule lists, ranked labels, sequences of steps)
without encoding it separately. See [`doc/comparison.md`](doc/comparison.md)
for how OGDL compares with JSON, YAML, TOML, XML, KDL and others, and when each
is the better choice.

```
server
  host db.example.com
  port 8080
user
  name Ana
  role admin
  role editor
```

That document is a tree of text, nothing more. `8080` is the name of a node,
not a typed number; types belong to whoever reads the tree.

New to OGDL? Start with **[the guide](doc/guide.md)**: the rationale, the
language by example, and how-tos for schemas and templates.

## Contents

The package is organised in five layers, each specified by an RFC-style
document in [`rfc/`](rfc/):

| Layer | What it does | Spec |
|---|---|---|
| **Core** | Parse a stream into a tree; serialize it back in canonical or compact form, with comments kept if wanted | [`ogdl-rfc.md`](rfc/ogdl-rfc.md) |
| **Path** | Address nodes: `user.role{1}`, `items[0]`, `host.'192.168.0.10'` | [`ogdl-path-rfc.md`](rfc/ogdl-path-rfc.md) |
| **Expressions** | Compute over a tree: `user.age >= 18 && user.name == 'Ana'` | [`ogdl-expr-rfc.md`](rfc/ogdl-expr-rfc.md) |
| **Templates** | Render text: `Hello $user.name`, `$if(...)`, `$for(x, ...)` | [`ogdl-template-rfc.md`](rfc/ogdl-template-rfc.md) |
| **Schema** | Validate a document, and optionally type its values: `port !int`, `rule* !action !string` | [`ogdl-schema-rfc.md`](rfc/ogdl-schema-rfc.md) |

Earlier informal descriptions of OGDL left large parts undefined. The RFCs here
pin those down, and [`rfc/analysis.md`](rfc/analysis.md) and
[`rfc/path-notes.md`](rfc/path-notes.md) record why each decision was made.

## Installation

```
go get github.com/rveen/ogdl/v2
```

It is a Go module requiring Go 1.21 or later, with no dependencies outside the
standard library. Version 1 (`github.com/rveen/ogdl`) stays available beside
it; [`doc/migration.md`](doc/migration.md) maps one onto the other.

## The language in brief

A line reads like a command line: **the first name is the head, and the names
after it are its children**, siblings of one another. An indented line attaches
to the head of the line above:

```
rule allow any        rule
  log yes               allow
                        any
                        log
                          yes
```

A line therefore holds one node and its leaf children, and each further level
takes a further line — the same rule at every depth. A record is one field per
line:

```
S1
  value 1.0
  class asil
  title "S1 – Slight injury"
```

`S1 value 1.0 class asil` on one line is `S1` with four flat children. `=` is
plain text, so `k=v`, `S>=S2`, `--flag=x` and URLs are single names that need
no quoting.

Other elements:

- **Quoted names**, `'...'` or `"..."`, may contain spaces and syntax
  characters. The only escapes are `\\`, `\'`, `\"`, `\n` and `\t`; any other
  is an error. A quoted string may span lines.
- **Comments** start with `#` at the start of a line or after a space, and must
  be followed by a space or line end.
- **Blocks**: a lone `\` at the end of a line makes the following, more-indented
  lines one verbatim text node.
- Indentation uses spaces or tabs, not both. The first indented line decides
  which, and a mix is rejected, as is a dedent to a level that was never
  opened, both as in Python.

Malformed input is always an error, reported as a `*SyntaxError` with line and
column. The parser never tries to repair input.

## Usage

### Parsing and serializing

```go
g, err := ogdl.ParseString(src) // or ogdl.Parse(r io.Reader)
if err != nil {
    var se *ogdl.SyntaxError
    if errors.As(err, &se) { /* se.Line, se.Col */ }
    if errors.Is(err, ogdl.ErrMixedIndentation) { /* ... */ }
}

text, err := g.Canonical() // canonical form: 2 spaces/level, one node per line
fmt.Print(g.Text())        // same, ignoring the error
text, err = g.Compact()    // compact form: leaf children inline, for people
```

`Parse` returns an unnamed root whose children are the top-level nodes. A node
is simply:

```go
type Node struct {
    Data interface{} // always a string after parsing; any Go value when built in code
    Out  []*Node     // ordered children
}
```

### Editing a file without losing its comments

Comments are not part of the tree, so `Parse` drops them. `ParseDocument`
keeps them alongside the tree, attached to the nodes they describe, together
with which nodes were written inline:

```go
d, err := ogdl.ParseDocumentString(src)
d.Root.Set("server.port", 9090)
out, err := d.Format() // comments, blank lines and layout preserved
```

A node keeps its comments for as long as the program keeps the node. New nodes
are laid out in compact form. `Format` normalizes indentation to two spaces and
writes blocks and multi-line strings as quoted strings; otherwise a file in
normal layout comes back byte for byte. `d.Comments[node]` gives read and write
access to a node's comments.

### Paths

```go
g.Get("server.port").String()     // "8080"   text below the node
g.Get("server.port").ThisString() // "port"   the node's own name
g.Get("server.port").Int64()      // 8080
g.Get("user.role{1}").String()    // "editor" second child named role
g.Get("user.role{}")              // a set of all role nodes
g.Get("user[0]")                  // first child of user, by position
```

`Get` never returns nil. When nothing matches, or the path is malformed, it
returns an empty node. Use `GetErr` to tell a malformed path apart from one
that simply matched nothing.

Path tokens consist of letters, digits and `_` only. Anything else must be
quoted, as in `iface.'eth0-1'`, so that `-` and the other operator characters
stay free for expressions.

### Expressions

```go
v, err := g.EvalString("server.port + 1")           // int64(8081)
v, err  = g.EvalString("user.name == 'Ana'")         // true
v, err  = g.EvalString("'4' + 3")                    // int64(7): text is coerced at the operator
v, err  = g.EvalString("user._len")                  // int64(3): built-in member
v, err  = g.EvalString("total = server.port * 2")    // assignment writes into the tree
```

The language has integers, floats, strings and booleans; arithmetic, comparison,
`&&`, `||`, `!`; assignment and compound assignment (`=`, `+=`, ...); indexes
computed by an expression, `a[i+1]`; indirection, `a.(b)`; and calls into Go
values stored in the tree:

```go
ctx := &ogdl.Node{}
ctx.Set("up", strings.ToUpper)
ctx.EvalString("up('hi')") // "HI"
```

Exported fields, methods and map keys of Go values are reachable by path. A
child node with the same name always takes precedence over a Go member. An
argument that cannot be converted to the parameter type is an error; it is
never silently replaced with a zero value.

> Because assignment exists, evaluation can modify the tree. When the
> expression comes from an untrusted source, evaluate it against a copy.

### Templates

```go
out, err := ogdl.ProcessString(`Hello $user.name, port ${server.port}
$if(user.role{1})
  has a second role
$else
  single role
$end
$for(r, user.role{})- $r.role
$end
Price: $$5`, g)
```

| Syntax | Meaning |
|---|---|
| `$path`, `${path}` | substitute a value |
| `$(expr)` | evaluate without output, e.g. for assignment |
| `$if(e)` `$elseif(e)` `$else` `$end` | conditionals |
| `$for(x, e)` ... `$end`, `$break` | loops over a node's children or a Go slice |
| `$$` | a literal `$` (and `$` followed by anything that cannot start a path is literal, so `$5.00` is left alone) |

A template is compiled once with `ParseTemplate` and then rendered with
`(*Template).Process(ctx)`. Errors are always returned. A path that matches
nothing renders as empty text, which is not an error. A loop variable hides
any node with the same name only while the loop runs, and that node is
restored afterwards. Text outside directives is copied byte for byte.

#### HTML

For web pages, use the escaping variants. Every substitution is HTML-escaped;
the template's own text, where the markup lives, is not:

```go
ctx.Set("comment", "<script>alert(1)</script>")
ogdl.ProcessHTMLString(`<p>$comment</p>`, ctx)
// <p>&lt;script&gt;alert(1)&lt;/script&gt;</p>
```

To write trusted markup unescaped, apply the built-in `_raw`:

```go
ogdl.ProcessHTMLString(`<article>$page.body._raw</article>`, ctx)
```

Go code can do the same by handing the template a value of type `ogdl.HTML`,
for example the result of a sanitizer. A document cannot: a node it names
`_raw` is just a node, and its content is still escaped. Use `_raw` only on
content from a trusted source.

`ParseHTMLTemplate` is the compiled form; setting `EscapeHTML` on any
`*Template` does the same. The escaping is not context-aware: it protects
element content and quoted attribute values, but not values placed inside
`<script>` or `<style>`, in unquoted attributes, or used as URLs, where
`javascript:` links remain possible. Validate data used in those places.

#### Performance

[`template_bench_test.go`](template_bench_test.go) renders the same HTML page
with OGDL (HTML mode) and with the standard library's `html/template`: escaped
substitutions, if / else-if / else, a numeric comparison, nested loops with a
break, and a length. A test checks that both produce byte-identical output.
OGDL renders from a tree of text; `html/template` from Go structs.

| Benchmark | OGDL | `html/template` |
|---|---|---|
| Render, 10 products | 38 µs · 13.5 KB · 595 allocs | 79 µs · 18.9 KB · 599 allocs |
| Render, 100 products | 363 µs · 133 KB · 5,637 allocs | 733 µs · 182 KB · 5,606 allocs |
| Render, 1000 products | 3.90 ms · 1.39 MB · 56,059 allocs | 7.74 ms · 1.88 MB · 56,366 allocs |
| Parse + first render | 28 µs · 11.3 KB · 239 allocs | 72 µs · 27.2 KB · 389 allocs |

Medians of 5 runs, per operation, Go 1.25.6, linux/amd64, Intel Core i5-9400
(recorded 2026-11-06). "Parse + first render" includes one execution against
an empty page, because `html/template` does its contextual escaping analysis
on first use. Reproduce with:

```
go test -run '^$' -bench 'Template(Execute|Parse)' -benchmem -count 5
```

### Schemas

A schema is an OGDL document that looks like the documents it describes, with
values replaced by types:

```go
s, err := ogdl.ParseSchema(`
!type action allow deny

server
  host !string
  port !int
  timeout? !float
  rule* !action !string
`)
err = s.Validate(g) // nil, or a *ValidationError listing every issue by path
err = s.Apply(g)    // validate, then turn !int/!float/!bool values into Go values
```

```
server.port[0]: "80x" is not !int
server.prot: unexpected; expected one of host, port, timeout, rule
```

The types are `!string`, `!int`, `!float`, `!bool` and `!any`, plus your own
enumerations and unions defined with `!type`. A name may end in `?`
(optional), `*` (any number) or `+` (at least one). Children the schema does
not mention are errors, which catches misspelled names; add `!any*` where
free-form content is wanted.

## Relationship to `github.com/rveen/ogdl`

This package is a re-implementation from specification, not a fork. It
intentionally differs from the earlier `rveen/ogdl` package in several places:
zero-based indexing and set semantics in paths, coercion of text to numbers,
errors where the old package returned nil or a zero value, scoped loop
variables (the old package deleted the node a loop variable shadowed), and no
`Graph` type. Each difference is listed in §12 of the expression RFC, §10 of
the template RFC, and `path-notes.md`.

## Development

```
go test ./...
```

The test vectors in each RFC's Appendix A are reproduced in the tests, for
example `TestVectors` and `TestVectorsIllFormed` in `parser_test.go`; the
comment attachment rule of the core RFC's Appendix B is covered in
`document_test.go`, and the schema RFC's Appendix A in `schema_test.go`. When you change an RFC, change the matching tests too.

## License

MIT. See [LICENSE](LICENSE).
