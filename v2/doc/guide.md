# The OGDL Guide

OGDL — the Ordered Graph Data Language — is a small, indentation-based text
format for **ordered trees**. This guide explains why it exists, teaches the
language by example, and then walks through the four layers built on it in
this Go package: paths, expressions, schemas and templates.

Every example in this guide is checked by `guide_test.go`, so what you read
here is what the code does.

- [1. A first look](#1-a-first-look)
- [2. Why OGDL](#2-why-ogdl)
- [3. The language](#3-the-language)
- [4. Examples](#4-examples)
- [5. Using OGDL from Go](#5-using-ogdl-from-go)
- [6. Paths and expressions](#6-paths-and-expressions)
- [7. Schemas: how to](#7-schemas-how-to)
- [8. Templates: how to](#8-templates-how-to)
- [9. Cheat sheet](#9-cheat-sheet)
- [10. Further reading](#10-further-reading)

---

## 1. A first look

```ogdl
# Firewall for the office network
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
```

There are no braces, no colons, no commas, no quotes. Indentation gives the
structure; each line names a node and, after it, that node's children. The
three `rule` lines are three nodes with the same name, kept in the order
written — which matters, because the first matching rule wins.

This one document is used as the running example throughout the guide.

---

## 2. Why OGDL

### One container instead of two

JSON, YAML and TOML give you two containers: **maps** (named, unordered,
unique keys) and **lists** (ordered, unnamed). Real data often needs both at
once — *an ordered sequence of named things* — and then you must encode one in
the other:

```json
{ "firewall": { "default": "deny",
    "rules": [ { "action": "allow", "from": "10.0.0.0/8", "port": 22 },
               { "action": "allow", "from": "192.168.0.0/16" },
               { "action": "deny",  "from": "any" } ] } }
```

The plural key `rules`, the field `action`, the array of one-entry objects:
all are conventions every reader must know.

OGDL has **one** container: at every level, an ordered list of named nodes.
Names may repeat and order always survives, so the rule list above needs no
encoding at all.

### Low noise

OGDL's other aim is to keep visible punctuation to a minimum. Structure comes
from indentation and line breaks; the only marks you will commonly type are
`#` for comments and quotes for text with spaces.

### No type guessing

`8080`, `no`, `0755`, `1e3` and `2026-10-07` are all just text. YAML's famous
surprises — `country: no` becoming `false` — cannot happen, because nothing
in the stream carries a type. Types are added deliberately, by a schema
([section 7](#7-schemas-how-to)) or by the program that reads the data.

### Equality you can compute

Every tree has exactly one **canonical** text form. Two documents mean the
same thing if and only if their canonical forms are byte-identical, which
makes hashing, signing, caching and diffing trivial.

### A small, safe core

No references, no anchors, no includes, no executable constructs. Parsing an
untrusted document cannot expand into gigabytes or run code. Errors are always
reported with line and column; the parser never guesses.

### When not to use it

- **Data exchange with other organisations**: use JSON. Ecosystem beats
  elegance, and few languages have an OGDL parser.
- **Flat, typed settings**: TOML says it with less ceremony.
- **Prose with markup**: use XML or HTML.

OGDL shines for ordered, hierarchical, human-maintained data — rule sets,
pipelines, ranked scales, taxonomies — processed by programs that use this
package. See [`comparison.md`](comparison.md) for the full comparison.

---

## 3. The language

### 3.1 Lines and indentation

A node is written on its own line; its children are indented below it:

```ogdl
network
  eth0
    ip 192.168.0.10
  eth1
    ip 192.168.1.10
```

Any amount of indentation works, as long as it is consistent: use spaces or
tabs, not both, and dedent only to a level you opened before. Both mistakes
are errors, exactly as in Python.

### 3.2 A line is a head and its arguments

The first name on a line is the **head**. The names after it are its
**children**, as siblings — the way a command line reads:

```ogdl
rule allow any          rule
                          allow
                          any
```

An indented line below attaches to the head, after the arguments:

```ogdl
rule allow any          rule
  log yes                 allow
                          any
                          log
                            yes
```

So `a b c` is `a` with two children, never a chain three levels deep. Going
deeper always takes a new line.

### 3.3 Records: one field per line

A line holds one node and its leaf children — never more. A record, whose
fields each have a value, therefore takes a line per field:

```ogdl
S1
  value 1.0
  class asil
  title "S1 – Slight injury"
```

> **The one thing to remember.** `S1 value 1.0 class asil` is `S1` with
> *four* flat children, not two fields. Each further level of structure is a
> further line, at every depth.

`=` has no special meaning: it is ordinary text, like every character other
than spaces, quotes, `#` and `\`. None of these need quoting:

```ogdl
when S>=S2
check a==b
args --verbose --level=3
url https://example.com/?q=ogdl
```

### 3.4 Quoting

Quote a name that contains spaces, or that would otherwise be read as
syntax. Single and double quotes are interchangeable:

```ogdl
title "Hello, world"
greeting 'She said "hi"'
path C:\Users\ana
```

Inside quotes the only escapes are `\\`, `\"`, `\'`, `\n` and `\t`; any other
backslash sequence is an error. Outside quotes a backslash is an ordinary
character, as in the Windows path above.

### 3.5 Multi-line text

A quoted string may span lines; the indentation of continuation lines is
dropped:

```ogdl
motto "two
       lines"
```

For longer text, end a line with a lone `\` — a **block**. The more-indented
lines below become one text node, verbatim, keeping relative indentation:

```ogdl
note \
  Line one
    indented
  Line three
```

### 3.6 Comments

`#` starts a comment at the beginning of a line, or after a space, when it is
followed by a space or the end of the line. Elsewhere `#` is text, so `#ff0000`
and `issue#12` are plain names.

```ogdl
# a comment
port 8080   # a trailing comment
color #ff0000
```

Comments are not data — they create no nodes — but the package can keep them
when a program edits a file ([section 5.4](#54-editing-a-file-and-keeping-its-comments)).

### 3.7 Errors

Ill-formed input is always rejected, with a line and column: mixed tabs and
spaces, a dedent to a level never opened, an unterminated quote, an undefined
escape, a control character. There is no lenient mode that guesses what you
meant.

---

## 4. Examples

### A ranked scale

Order is meaning: a severity's rank is its position.

```ogdl
severity
  S0 "No injuries"
  S1 "Light injuries"
  S2 "Severe injuries"
  S3 "Fatal injuries"
```

### A build pipeline

Steps run in order; names repeat freely.

```ogdl
pipeline
  step checkout
    repo git@example.com:app.git
  step run
    cmd "go vet ./..."
  step run
    cmd "go test ./..."
  step publish
    when tag
```

### A map with free names

Host names are data, not schema; each host has the same shape.

```ogdl
hosts
  web
    ip 10.0.0.1
    role frontend
  db
    ip 10.0.0.2
    role storage
    backup daily
```

### A menu tree

Arbitrary depth, one line per level.

```ogdl
menu
  File
    New
    Open
    Recent
      report.ogdl
  Edit
    Undo
```

---

## 5. Using OGDL from Go

```sh
go get github.com/rveen/ogdl/v2
```

### 5.1 Parsing

```go
g, err := ogdl.ParseString(src) // or ogdl.Parse(r io.Reader)
if err != nil {
    var se *ogdl.SyntaxError // se.Line, se.Col
    ...
}
```

`g` is an unnamed root node whose children are the top-level nodes. A node is
just data and an ordered list of children:

```go
type Node struct {
    Data interface{} // a string after parsing; any Go value when built in code
    Out  []*Node
}
```

### 5.2 Reading values

The text *below* a node is its value; the node's own text is its name:

```go
g.Get("firewall.default").String()      // "deny"
g.Get("firewall.default").ThisString()  // "default"
g.Get("firewall.rule.port").Int64()     // 22, from the first rule
g.Get("firewall.rule{1}.from").String() // "192.168.0.0/16"
```

`Get` never returns nil: when nothing matches it returns an empty node, whose
`String()` is `""` and whose `Int64()` is 0 (or a default you pass). `Int64`,
`Float64`, `Bool` and `Strings` convert the text below a node.

### 5.3 Writing

Three serializations, for three purposes:

| Method | Form | Use it for |
|---|---|---|
| `Canonical()` / `Text()` | one node per line, two spaces | equality, hashing, diffs |
| `Compact()` | leaf children on the head's line | output people read |
| `Document.Format()` | the author's own layout, with comments | rewriting a file |

The running example in compact form:

```text
firewall
  default deny
  rule allow
    from 10.0.0.0/8
    port 22
  rule allow
    from 192.168.0.0/16
  rule deny
    from any
```

Only leaves go on the head's line: `default deny`, `rule allow` and
`from any` are inline, while the fields of a rule, which have values of their
own, take a line each.

### 5.4 Editing a file and keeping its comments

`Parse` drops comments. `ParseDocument` keeps them, attached to the nodes
they describe, together with which nodes were written inline:

```go
d, _ := ogdl.ParseDocumentString(src)
d.Root.Set("firewall.default", "allow")
fw := d.Root.S("firewall")
fw.Out = append(fw.Out, &ogdl.Node{Data: "rule", Out: []*ogdl.Node{
    {Data: "log"}, {Data: "from", Out: []*ogdl.Node{{Data: "any"}}},
}})
out, _ := d.Format()
```

```ogdl
# Firewall for the office network
firewall
  default allow
  # first match wins
  rule allow
    from 10.0.0.0/8
    port 22
  rule allow
    from 192.168.0.0/16
  rule deny
    from any
  rule log
    from any
```

A node keeps its comments for as long as your program keeps the node; new
nodes are laid out compactly.

---

## 6. Paths and expressions

### 6.1 Paths

A path walks the tree by name:

| Path | Selects |
|---|---|
| `firewall.default` | the first child named `default` of `firewall` |
| `firewall.rule{1}` | the second `rule` (counting from zero) |
| `firewall.rule{}` | all `rule` nodes, as a set |
| `firewall.rule[0]` | the first *child* of the first `rule`: `allow` |
| `hosts.'web-1'` | a name with characters outside letters, digits and `_` |

`{n}` counts nodes with the preceding name; `[n]` counts all children by
position. Path tokens are letters, digits and `_` only — anything else must
be quoted, because the other characters belong to expressions.

### 6.2 Expressions

`EvalString` computes over the tree:

```go
g.EvalString("firewall.rule.port + 1")             // int64(23)
g.EvalString("firewall.default == 'deny'")         // true
g.EvalString("firewall.rule{2}[0]")                // "deny"
```

Text becomes a number at the operator that needs one, so `'4' + 3` is `7`.
There are integers, floats, strings and booleans; arithmetic, comparisons,
`&&`, `||`, `!`; assignment (`=`, `+=`, …); and calls into Go functions stored
in the tree. A few built-in members compute facts about a node: `_len` (number
of children), `_name` (the node's own name), `_string`, `_this` and `_raw`.

Because assignment exists, evaluating an expression can change the tree.
Evaluate untrusted expressions against a copy.

---

## 7. Schemas: how to

A schema says which nodes a document may contain, how often, and what their
values look like. It is itself an OGDL document — the data with its values
replaced by types.

### 7.1 Write the schema from an example

Start from a real document and replace each value by its type:

```ogdl
!type action allow deny

firewall
  default !action
  rule+ !action
    from !string
    port? !int
```

Read it as: `firewall` has exactly one `default`, whose value is `allow` or
`deny`; one or more `rule` nodes, each with an action as its first child, one
`from` and an optional integer `port`.

### 7.2 Types

| Type | Accepts |
|---|---|
| `!string` | any text, including empty |
| `!int` | a 64-bit integer: `42`, `-7`, `+3` |
| `!float` | a decimal number: `1`, `2.5`, `-1e3`, `.5` |
| `!bool` | exactly `true` or `false` |
| `!any` | any node, with any children (not checked) |

`yes`, `on`, `0x10` and `NaN` are not booleans or numbers; the schema
introduces no implicit typing.

Define your own with `!type`:

```ogdl
!type action allow deny          # an enumeration
!type number !int !float         # a union
!type limit !int unlimited       # a union with a literal
!type port !int                  # an alias
```

### 7.3 How often: `?`, `*`, `+`

| Mark | Meaning |
|---|---|
| (none) | exactly one |
| `?` | optional |
| `*` | any number |
| `+` | at least one |

The mark goes on the name: `port? !int`, `rule* !action`, `!string*`.

> `=` means nothing in OGDL, so `port=!int` is one word, not a field with a
> type. The schema compiler rejects an unescaped `=` in a name and tells you
> to write `port !int`.

### 7.4 Three shapes you will meet

**Named fields** match by name, in any order:

```ogdl
server
  host !string
  port !int
```

**Positional values** — the arguments of a line — are matched in order by
types:

```ogdl
rule* !action !string        # rule allow 10.0.0.0/8
```

**Maps** — free names, fixed content — use a type in place of the name:

```ogdl
hosts
  !string*
    ip !string
    role? !string
```

Fields and positional values mix freely: named fields are taken first, by
name, and the remaining children are matched in order by the types.

### 7.5 Closed by default

A child the schema does not mention is an error. That is what catches
`prot 8080` for `port 8080`. Where free-form content is wanted, say so with
`!any*`:

```ogdl
metadata? !any*
```

### 7.6 Validate

```go
s, err := ogdl.ParseSchema(schemaText) // *SchemaError if the schema is wrong
err = s.Validate(g)                    // nil, or *ValidationError
```

Every problem is reported, by path. For this document

```ogdl
firewall
  default drop
  rule allow
    from any
    port ssh
  rule alow
    form x
```

the issues are:

```text
firewall.default[0]: "drop" is not !action ("allow", "deny")
firewall.rule.port[0]: "ssh" is not !int
firewall.rule{1}: missing from
firewall.rule{1}[0]: "alow" is not !action ("allow", "deny")
firewall.rule{1}.form: unexpected; expected one of from, port
```

A value with the wrong type is addressed by position (`[0]`); everything else
by name.

### 7.7 Apply: turn text into typed values

`Apply` validates and then replaces the text of every `!int`, `!float` and
`!bool` value with an `int64`, `float64` or `bool`:

```go
if err := s.Apply(g); err == nil {
    port := g.Get("firewall.rule.port[0]").Data.(int64) // 22
}
```

An invalid document is left untouched. Typed values still serialize as the
same text.

### 7.8 One subtlety: no backtracking

Types match greedily and never give a value back. With `args !int* !string`,
`args 1 2 x` is valid, but `args 1 2 3` is not: `!int*` takes all three
numbers and nothing is left for `!string`. If you need the other reading,
name the last value instead of typing it.

---

## 8. Templates: how to

A template is text — plain text, HTML, a config file, anything — in which `$`
switches to the expression language. Everything else is copied byte for byte.

```go
out, err := ogdl.ProcessString(tpl, g)       // plain text
out, err := ogdl.ProcessHTMLString(tpl, g)   // HTML: substitutions escaped
t, err := ogdl.ParseTemplate(tpl)            // compile once, then t.Process(g)
```

### 8.1 Substitution

```template
Default policy: $firewall.default
```

renders `Default policy: deny`. A path that matches nothing renders as
nothing; that is not an error.

Use braces to end a path where text continues: `${name}s`. Braces are also
needed before a dot that is just punctuation — `$n.` reads as a path ending
in `.`, which is an error, so write `${n}.`.

### 8.2 Conditionals

```template
$if(firewall.default == 'deny')Closed by default.$else Open by default.$end
```

`$elseif(...)` is available too. A condition that is a missing path is
false; any non-empty value is true.

### 8.3 Loops

`$for(var, expr)` iterates the children of a node, or the members of a set:

```template
$for(r, firewall.rule{})$r[0] $r.from$if(r.port) port $r.port$end
$end
```

renders

```text
allow 10.0.0.0/8 port 22
allow 192.168.0.0/16
deny any
```

On each iteration `r` is the current element — here a `rule` node — so
`$r.from` is its source and `$r[0]` its first child, the action. `$r._name` is
the element's own name, `rule`. Iterating `firewall` instead of
`firewall.rule{}` would also visit `default`.

The loop variable exists only inside the loop, and is never written into the
tree: while the loop runs it hides a node with the same name, which is
untouched and visible again afterwards.

`$break` leaves the innermost loop:

```template
$for(r, firewall.rule{})$if(r[0] == 'deny')$break$end$r.from
$end
```

renders the sources up to the first `deny`:

```text
10.0.0.0/8
192.168.0.0/16
```

### 8.4 Computing: `$(...)`

`$(expr)` evaluates without writing anything — for assignments:

```template
$(n = 0)$for(r, firewall.rule{})$(n += 1)${n}. $r[0] $r.from
$end
```

```text
1. allow 10.0.0.0/8
2. allow 192.168.0.0/16
3. deny any
```

Assignments write into the context tree. If that matters, process the
template against a copy.

### 8.5 Dollar signs

`$$` writes a literal `$`. A `$` that cannot start anything — followed by a
digit, a space, punctuation — is written as is, so `Price: $5.00` needs no
escaping.

### 8.6 HTML

For web pages, use the HTML variants. Every substitution is escaped; the
template's own markup is not:

```template
<ul>
$for(m, team.member{})  <li>$m.name$if(m.role == 'lead') (lead)$end</li>
$end</ul>
```

with

```ogdl
team
  member
    name Ana
    role lead
  member
    name Bo
    role dev
  member
    name "<Eve>"
    role dev
```

renders

```html
<ul>
  <li>Ana (lead)</li>
  <li>Bo</li>
  <li>&lt;Eve&gt;</li>
</ul>
```

To insert trusted markup unescaped, apply the built-in `_raw`:
`$page.body._raw`. A document cannot do this on its own — a node named
`_raw` is just a node — so only template authors and Go code decide what is
trusted. Escaping protects element content and quoted attributes, not
`<script>`, unquoted attributes or URLs.

### 8.7 Errors

Templates report errors instead of producing partial output: an unclosed
`$if`, an `$end` without an opening, a `$break` outside a loop, a malformed
path, a division by zero. Only a path that matches nothing is silent.

---

## 9. Cheat sheet

**Language**

| Write | Means |
|---|---|
| `a b c` | `a` with children `b`, `c` |
| `a k=v` | `a` with one child, `k=v` — `=` is plain text |
| indented line | children of the head above |
| `"…"`, `'…'` | a name with spaces or syntax characters |
| `\` at end of line | the indented lines below are one text node |
| `# …` | a comment (at line start or after a space) |

**Paths**: `a.b` by name · `a.b{1}` second `b` · `a.b{}` all `b` ·
`a[0]` first child · `a.'x-y'` quoted.

**Schema**: `!string !int !float !bool !any` · `!type name alt…` ·
`name?` `name*` `name+` · `!any*` to allow anything.

**Templates**: `$path` `${path}` `$(expr)` · `$if(…)` `$elseif(…)` `$else`
`$end` · `$for(x, …)` `$break` `$end` · `$$`.

---

## 10. Further reading

| Topic | Specification |
|---|---|
| The language | [`rfc/ogdl-rfc.md`](../rfc/ogdl-rfc.md) |
| Paths | [`rfc/ogdl-path-rfc.md`](../rfc/ogdl-path-rfc.md) |
| Expressions | [`rfc/ogdl-expr-rfc.md`](../rfc/ogdl-expr-rfc.md) |
| Schemas | [`rfc/ogdl-schema-rfc.md`](../rfc/ogdl-schema-rfc.md) |
| Templates | [`rfc/ogdl-template-rfc.md`](../rfc/ogdl-template-rfc.md) |
| OGDL among other formats | [`comparison.md`](comparison.md) |
| Design decisions | [`rfc/analysis.md`](../rfc/analysis.md) |
