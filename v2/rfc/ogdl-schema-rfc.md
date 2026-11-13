# RFC: OGDL Schema

## Status of This Memo

This document specifies a schema language for OGDL: a way to state which nodes a document may contain, how often, and what text their values may hold, and to check a document against that statement. It is implemented by `schema.go` in this repository.

The key words MUST, MUST NOT, REQUIRED, SHALL, SHALL NOT, SHOULD, SHOULD NOT, RECOMMENDED, MAY, and OPTIONAL in this document are to be interpreted as described in RFC 2119.

It builds on the core specification (`ogdl-rfc.md`) and uses the path specification (`ogdl-path-rfc.md`) to report where a document fails.



## Abstract

An OGDL schema is itself an OGDL document. It mirrors the shape of the documents it describes: a name in the schema stands for the same name in the document, and a **type**, written `!int`, `!bool` and so on, stands for any name of that type. Occurrence is marked with `?`, `*` or `+` after a name. There are five built-in types — `string`, `int`, `float`, `bool` and `any` — and a schema can define more as sets of allowed names, with `!type`. Validation reports every mismatch by path, and can optionally convert matched values from text to typed values.



## 1. Introduction

OGDL deliberately carries no types: `8080` is the name of a node, and whether it is a number is for the reader to decide (core specification, section 2.2). The cost is that every consumer decides separately, in code, and that a typo in a hand-written file — `prot 8080` — goes unnoticed until something fails to find `port`.

A schema moves those decisions into one place, written in the same language as the data:

```
!type action allow deny

server
  host !string
  port !int
  timeout? !float
  debug? !bool
  rule* !action !string
```

describes

```
server
  host db.example.com
  port 8080
  debug true
  rule allow 10.0.0.0/8
  rule deny any
```

The design goals are:

* **A schema is an OGDL document.** It needs no parser of its own, and it can be read, written, queried and templated with the same tools as data.
* **The schema looks like the data.** A schema for a document is that document with values replaced by types.
* **Just enough types.** Five built-in types, plus enumerations and unions. Anything richer — dates, sizes, patterns, ranges — is left out of this version, with room reserved for it (section 10).
* **Closed by default.** A child the schema does not mention is an error, because the most common mistake in a hand-written file is a misspelled name.
* **Deterministic and linear.** Matching never backtracks.



## 2. Schema Documents

A schema is an OGDL document (core specification). It is parsed by a conforming OGDL parser and interpreted as a schema afterwards; nothing in this document changes how a stream is parsed.

The nodes of a schema are of three kinds, told apart by their names (section 3):

* **Named particles** — a literal name, optionally followed by an occurrence mark. `port`, `timeout?`, `rule*`. A named particle matches a document node with exactly that name.
* **Type particles** — `!` followed by a type name, optionally followed by an occurrence mark. `!int`, `!string*`. A type particle matches a document node whose name is a value of that type.
* **Type definitions** — the directive `!type` at depth zero of the schema, followed by the new type's name and its alternatives. `!type action allow deny`. A definition creates a type; it matches nothing by itself.

The depth-zero particles of a schema — every depth-zero node except definitions — describe the children of the document's root node. The children of a particle in the schema describe the children of the document nodes it matches.

How a document identifies its schema is outside the scope of this specification. A document carries no reference to a schema; the application supplies both.



## 3. Names in a Schema

### 3.1 Directives

A schema name that begins with `!` refers to a type. The rest of the name, after removing an occurrence mark (section 3.2), MUST be a type name:

```
type_name = letter ( letter | digit | "_" | "-" )*
```

`letter` is any character of Unicode General Category L, and `digit` any character of category Nd.

A name consisting of `!` alone, or `!` followed by something that is not a type name, is a schema error (section 8). Names beginning with `!` are thereby reserved, which leaves room for future directives (section 10).

`type` is not a type name but the one directive this version defines: `!type` introduces a definition (section 4.2). It cannot be defined, referred to, or used as a particle.

### 3.2 Occurrence

The last character of a particle's name, if it is one of `?`, `*` or `+`, is an occurrence mark and is not part of the name:

| Mark | Occurrences |
|---|---|
| none | exactly one |
| `?` | zero or one |
| `*` | zero or more |
| `+` | one or more |

`timeout?` is the particle `timeout`, optional; `!string*` is the type particle `!string`, any number of times.

`!type` takes no occurrence mark, and neither does the name of the type it defines; either is a schema error.

### 3.3 Escapes

Within schema names, and only there, `\` escapes the character that follows it. This is how a schema mentions a document name that would otherwise read as syntax:

| Schema name | Matches the document name |
|---|---|
| `\!important` | `!important` |
| `size\*` | `size*` |
| `a\\b` | `a\b` |

A `\` at the end of a name is a schema error. Note that the core specification already gives `\` no meaning inside words, so these names reach the schema layer exactly as written.

### 3.4 `=` in a Schema

A schema name that contains an unescaped `=` is a schema error. The core specification gives `=` no meaning, so `port=!int` is one word, and treating it as a literal name would silently describe a node nobody writes; an earlier draft of the core specification read it as `port` with the child `!int`, and schemas written for that draft must fail loudly rather than change meaning. The error says what was meant: write `port !int`. The rule also holds `=` free for constraints (section 10). A document name containing `=` is matched by escaping it: `a\=b`.



## 4. Types

A type is a predicate on a node's **name**. It does not by itself say anything about the node's children; those are described by the particle's children in the schema (section 5.3).

### 4.1 Built-in Types

| Type | Matches a name that is |
|---|---|
| `!string` | anything, including the empty name |
| `!int` | an integer that fits in 64 bits, signed |
| `!float` | a decimal number, integer or not |
| `!bool` | `true` or `false` |
| `!any` | anything; the node may also have any children, which are not checked |

The textual forms are exact. No whitespace is trimmed, no case is folded, and nothing else is accepted:

```
int   = [ "+" | "-" ] DIGIT+
float = [ "+" | "-" ] ( DIGIT+ [ "." DIGIT* ] | "." DIGIT+ ) [ exponent ]
exponent = ( "e" | "E" ) [ "+" | "-" ] DIGIT+
bool  = "true" | "false"
```

`DIGIT` is U+0030..U+0039. An `int` whose value does not fit in a signed 64-bit integer does not match `!int`; it matches `!float`. `float` covers every `int`. Hexadecimal forms, digit separators, `Inf` and `NaN` are not numbers. In particular `yes`, `no`, `on`, `off`, `1` and `0` are not booleans: OGDL has no implicit typing, and a schema does not introduce any.

These forms are chosen to agree with the expression language's coercions (expression RFC, section 2.3), so that a value a schema accepts as `!int` is one that `Int64()` and arithmetic read as the same integer.

### 4.2 Defined Types

A depth-zero schema node `!type` defines a type. Its first child is the new type's name, written without `!`. Its remaining children are the **alternatives**: each is either a literal name or a type reference, and the defined type matches a name if any alternative does. Written on one line, a definition reads as a head with arguments (core specification, section 6.5).

An enumeration lists literal names:

```
!type action allow deny
!type class asil ap
```

A union lists types, and may mix them with literals:

```
!type number !int !float
!type limit !int unlimited
```

An alias has one alternative:

```
!type port !int
```

A long enumeration may continue on indented lines, one alternative per line, since each attaches to the head. Two names on an indented line would make the second a child of the first, which an alternative may not have:

```
!type weekday mon tue wed
  thu
  fri
```

Names and alternatives are names only: they MUST NOT have children, and MUST NOT carry occurrence marks; either is a schema error.

The following are also schema errors:

* `!type` with no name, or with a name but no alternatives.
* `!type` below depth zero. There it would be read as a type particle, so it is rejected rather than reinterpreted.
* Defining one of the built-in types, or defining a type twice.
* Referring to a type that is neither built in nor defined.
* A cycle of definitions, such as `!a !b` together with `!b !a`.
A definition may appear anywhere among the depth-zero nodes; references to it may precede it.



## 5. Matching

Validation checks the document's root node against the schema's depth-zero particles, and then each matched node against the children of its particle, recursively.

### 5.1 Matching the Children of a Node

Let `N` be a document node and `P` the list of particles describing its children (the schema's depth-zero particles for the root, or a particle's children otherwise). `P` divides into its named particles and its type particles, the latter kept in schema order.

1. **Named particles first.** Each child of `N` whose name equals the name of a named particle in `P` is matched by that particle. Afterwards, the number of children each named particle matched MUST be within its occurrence (section 3.2). Named particles are matched by name, so the children they match may appear in any order and interleaved with others.
2. **Then type particles, in sequence.** The children of `N` not matched in step 1 are taken in document order and matched against the type particles in schema order. Each type particle takes as many consecutive children as match its type, up to its maximum occurrence, and never gives one back; it then MUST have taken at least its minimum occurrence. Matching then moves on to the next type particle.
3. **Nothing left over.** A child of `N` matched by neither step is an error: the schema is closed. To allow unknown children, end `P` with `!any*`.

A child whose name equals a named particle's name is never offered to a type particle, even if the named particle has already reached its maximum occurrence; a second `port` where only one is allowed is reported as too many `port`, not as an unexpected child.

Two names in `P` that are the same named particle, after removing occurrence marks and escapes, are a schema error.

Step 2 does not backtrack. With `args !int* !string`, the children `1 2 x` match, but `1 2 3` do not: `!int*` takes all three, and `!string` finds nothing. This is deliberate. It keeps matching linear, and it makes the result predictable from reading the schema left to right. A schema that needs the other reading can say so by naming the last child instead of typing it.

### 5.2 The Shape of a Typical Line

Steps 1 and 2 together fit the way OGDL lines are written: a head, its arguments, and named children below it.

```
rule* !action !string
  log? !bool
```

matches

```
rule allow 10.0.0.0/8
  log true
```

because `log` is taken by name, and `allow` and `10.0.0.0/8` are taken in order by the two type particles.

### 5.3 Matching a Node Against Its Particle

A document node matched by a particle is then checked against the particle's own children, by section 5.1, with one rule for particles that have none:

* A particle with **no children** in the schema matches only a **leaf**. `port !int` requires `port` to have exactly one child, an integer, with no children of its own; `enabled?` describes a flag, present or absent, with no value.
* The type `!any` is the exception: its node may have any children, and they are not checked.

A type particle with children describes nodes whose names are free but whose content is fixed — the pattern of a map:

```
hosts
  !string*
    ip !string
    alias* !string
```

matches any number of hosts, each named freely, each with one `ip` and any number of `alias` values.



## 6. Validation Result

Validation succeeds if and only if every node of the document is matched (section 5) and every count is within its occurrence.

A validator SHOULD report every error it finds rather than stopping at the first. Each error report MUST include:

* the **path** of the node the error concerns, written in path syntax (path RFC). A child whose name failed a type particle is identified by position, `[k]`, because its name is the value in question; any other child is identified by name, with `{n}` to distinguish repeated names. For example, `server.rule{1}[0]` is the first child of the second `rule`, and `server.prot` an unexpected child. A missing or surplus node is reported at its parent's path;
* what was expected, in schema terms — `!int`, `port`, `at least one rule` — and what was found.

For example, validating

```
server
  host db.example.com
  port 80x
  prot 8080
  rule allow 10.0.0.0/8
  rule maybe any
```

against the schema of section 1 reports, in some order:

```
server.port[0]: "80x" is not !int
server.prot: unexpected; expected one of host, port, timeout, debug, rule
server.rule{1}[0]: "maybe" is not !action ("allow", "deny")
```

An implementation that knows the line and column of a node (for example from a document that retains its layout) SHOULD include them as well.



## 7. Typing

A schema can do more than accept or reject: once a document is valid, it says which values are numbers and booleans. A validator MAY offer to **apply** a schema: after successful validation, the name of every node matched by `!int`, `!float` or `!bool` is replaced by the typed value it denotes — a 64-bit integer, a 64-bit float, or a boolean.

For a defined type, the conversion is that of the first alternative, in definition order, that matches: under `!type number !int !float`, `5` becomes an integer and `5.5` a float. A literal alternative, and the types `!string` and `!any`, convert nothing.

Applying a schema MUST NOT change a document that fails validation. Typing does not change the document's text: a typed value serializes to its textual form (core specification, section 8.2), so `port 8080` is still written `port 8080`. Whether the conversion produces exactly the same text — `1.50` read as `!float` is written back as `1.5` — is a property of the serializer, and an application that needs the original text should not apply the schema to the document it will write back.

Typing is optional. Validation alone gives every guarantee of section 6, and the expression layer already coerces text to numbers at its operators.



## 8. Schema Errors

A schema that is not well-formed MUST be rejected before any document is validated against it. The schema errors are:

* A name of `!` alone, or `!` followed by something other than a type name (section 3.1); `!type` used as a type.
* A `\` at the end of a name (section 3.3).
* An unescaped `=` in a name (section 3.4).
* A definition with an occurrence mark, without a name or alternatives, or whose name or alternatives have children or occurrence marks (section 4.2).
* A definition of a built-in type, a type defined twice, a reference to an undefined type, or a cycle of definitions (section 4.2).
* `!type` below depth zero (section 4.2).
* The same named particle twice among the children of one node (section 5.1).

The schema itself must, of course, be a well-formed OGDL document; errors of the core specification are reported as such.



## 9. Security Considerations

Validation is linear in the size of the document times the size of the schema: section 5.1 never backtracks, and type definitions form no cycles. A schema from an untrusted source can therefore not make validation run long, though a large schema still costs in proportion to its size.

A schema does not make a document safe to use. It checks shape and the text of values; it does not check that a host name resolves, that a path is inside a sandbox, or that a number is in range (section 10). Applications SHOULD continue to validate what they act on.

Applying a schema (section 7) changes node data in place. As with expression evaluation, an application that must keep the original tree should apply the schema to a copy.



## 10. Extensibility

This version defines five types and no constraints beyond enumeration. The following are reserved for a future version, and their syntax is held free by section 3.1, which makes every unknown `!` name an error today:

* **Constraints on built-in types**, possibly as `name=value` words on a definition: `!type port !int min=1 max=65535`, `!type code !string pattern=[A-Z]{3}`. Such words contain an unescaped `=` and are schema errors today (section 3.4), so adding constraints cannot change the meaning of a valid schema.
* **Further types**, such as `!date`, `!time` or `!duration`. Until then, an application can validate these as `!string` and parse them itself.
* **Named structures and recursion**, so that a subtree's shape can be reused, and arbitrarily deep trees — menus, taxonomies — described. Types in this version describe names only.
* **Ordered named particles**, for formats in which `host` must come before `port`. Order among repeated nodes of one name is already preserved by OGDL itself; this version does not constrain order among different names.



## 11. Conformance

A conforming validator:

* MUST reject every schema error of section 8, and MUST NOT validate a document against an ill-formed schema.
* MUST accept exactly the documents section 5 accepts, using exactly the textual forms of section 4.1.
* MUST report errors as section 6 requires.
* MAY offer typing (section 7); if it does, it MUST NOT apply it to an invalid document.
* MUST reproduce Appendix A.



## Appendix A. Test Vectors

Each case gives a schema, a document, and the expected result. Error paths are listed; the wording of messages is not normative.

### A.1 Values and occurrence

Schema:

```
server
  host !string
  port !int
  timeout? !float
  debug? !bool
```

| Document | Result |
|---|---|
| `server` / `··host a` / `··port 80` | valid |
| `server` / `··port 80` / `··host a` / `··timeout 1.5` / `··debug true` | valid: named particles in any order |
| `server` / `··host a` / `··port 80` / `··timeout 2` | valid: `!float` accepts integers |
| `server` / `··host a` | error at `server`: missing `port` |
| `server` / `··host a` / `··port 80` / `··port 81` | error at `server`: too many `port` |
| `server` / `··host a` / `··port 80.0` | error at `server.port[0]` |
| `server` / `··host a` / `··port 80` / `··debug yes` | error at `server.debug[0]`: `yes` is not `!bool` |
| `server` / `··host a` / `··port 99999999999999999999` | error at `server.port[0]`: does not fit in 64 bits |
| `server` / `··host a` / `··port 80` / `··extra 1` | error at `server.extra`: unexpected |
| `server` / `··host ""` / `··port -1` | valid |
| `server host=a port=80` | error at `server`: missing `host`, missing `port`; unexpected `host=a` and `port=80`, each one name |

### A.2 Arguments and named children

Schema:

```
!type action allow deny
rule* !action !string
  log? !bool
```

| Document | Result |
|---|---|
| (empty) | valid: `rule*` allows zero |
| `rule allow any` | valid |
| `rule deny 10.0.0.0/8` / `··log true` | valid |
| `rule allow` | error at `rule`: missing `!string` |
| `rule permit any` | error at `rule[0]`: `permit` is not `!action` |
| `rule allow any extra` | error at `rule.extra`: unexpected |
| `rule allow any log true` | error at `rule.log`: missing `!bool`; at `rule.true`: unexpected (arguments are leaves, so `log` has no child) |

### A.3 Maps

Schema:

```
hosts
  !string*
    ip !string
```

| Document | Result |
|---|---|
| `hosts` | valid |
| `hosts` / `··web` / `····ip 10.0.0.1` / `··db` / `····ip 10.0.0.2` | valid |
| `hosts` / `··web` | error at `hosts.web`: missing `ip` |

### A.4 No backtracking

Schema:

```
args !int* !string
```

| Document | Result |
|---|---|
| `args 1 2 x` | valid |
| `args 1 2 3` | error at `args`: missing `!string` (section 5.1) |

### A.5 Leaves, flags and `!any`

Schema:

```
enabled?
port !int
meta? !any*
```

| Document | Result |
|---|---|
| `enabled` / `port 1` | valid |
| `enabled yes` / `port 1` | error at `enabled.yes`: unexpected (a flag is a leaf) |
| `port` / `··1 x` | error at `port.1.x`: unexpected (a particle without children matches a leaf) |
| `port 1` / `··x 2` | error at `port.x`: unexpected (`x` attaches to the head `port`, after `1`) |
| `port 1` / `meta` | valid: `!any*` admits zero children |
| `port 1` / `meta a b` / `··c d` / `····e` | valid: `!any` admits any subtree, and `*` any number of them |

### A.6 Escapes

Schema:

```
\!important? !bool
size\* !int
```

| Document | Result |
|---|---|
| `!important true` / `size* 3` | valid |
| `size 3` | error at root: missing `size*`; unexpected `size` |

### A.7 Schema errors

Each of the following schemas MUST be rejected:

| Schema | Reason |
|---|---|
| `port=!int` | unescaped `=` (section 3.4) |
| `port?=!int` | unescaped `=` (section 3.4) |
| `a !` | `!` is not a type (section 3.1) |
| `a !integer` | undefined type |
| `a !type` | `!type` is not a type |
| `!type int !string` | redefines a built-in type |
| `!type a !b` / `!type b !a` | cycle of definitions |
| `!type* a x` | occurrence mark on `!type` |
| `!type a* x` | occurrence mark on a defined name |
| `!type a` | definition with no alternatives |
| `!type` | definition with no name |
| `a !int` / `a !string` | named particle `a` twice |
| `a` / `··!type t x` | definition below depth zero |
| `a\` | `\` at the end of a name |

### A.8 Root maps and typing

Schema:

```
!type number !int !float
!string*
  n !number
  on !bool
```

| Document | Result |
|---|---|
| `a` / `··n 1` / `··on true` / `b` / `··n 2.5` / `··on false` | valid; applying the schema makes `a.n[0]` the integer 1, `b.n[0]` the float 2.5, and the `on` values booleans |
| `a` / `··n 0x10` / `··on true` | error at `a.n[0]`: hexadecimal is not a number |



## Appendix B. Rationale (Informative)

**Why `!type`.** A definition could have been written `!action allow deny`, but at depth zero that is indistinguishable from a type particle `!action` with a child `allow` — and a root whose children are freely named (`!string*` at depth zero) is a reasonable thing to describe. A single reserved directive removes the ambiguity everywhere, and on one line it still reads as a head with arguments.

**Why `!`.** The schema needs one character to tell "this name" from "any name of this kind". `!` is rare at the start of real names, reads as a tag (as in YAML's `!!int`, but without YAML's implicit resolution), and has no meaning in OGDL streams, so a schema stays a plain document. It also continues the notation of an earlier OGDL type proposal, which used `!int` and `!bool`.

**Why these five types.** They are the scalar types every consumer of configuration already needs, and the ones the expression layer already understands: text, integer, float, boolean. `!any` is the escape hatch that makes a closed schema usable for partly free-form data. Dates, durations, sizes, addresses and patterns each come in several conventions; choosing one would make the schema language larger than the data language. Enumerations and unions cover closed vocabularies, which are the most common "type" in configuration after numbers.

**Why types describe names.** In OGDL a value is a node, so "the value of `port` is an integer" means "the child of `port` has an integer name". Making types predicates on names, placed where the value node goes, lets the schema have the same shape as the data, and lets the same mechanism describe arguments (`rule !action !string`), values (`port !int`) and map keys (`hosts` / `!string*`).

**Why named particles match in any order.** OGDL preserves order, and order matters among repeated nodes of one name — rule lists, ranked labels. It rarely matters between different names, and requiring `host` before `port` would make schemas brittle for no gain. Type particles, by contrast, describe positions, so they match in order.

**Why closed by default.** In a hand-written file, an unknown name is almost always a misspelling. `!any*` reopens a node explicitly where free-form content is wanted.

**Why no backtracking.** A regular-expression matcher would accept a few more schemas, at the cost of an exponential worst case and of results that are hard to predict by reading the schema. The possessive rule of section 5.1 is the one people apply when they read a schema left to right.

**Agreement with the value layer.** Go's `strconv.ParseFloat` also accepts hexadecimal floats such as `0x1p4`. Section 4.1 does not, and neither does the expression language, whose value layer applies the same decimal grammar when it coerces text to a number. A value `!float` accepts is therefore exactly a value arithmetic reads as a number.
