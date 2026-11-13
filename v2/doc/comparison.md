# OGDL compared with other data languages

This document places OGDL among the data languages it competes with — JSON,
YAML, TOML, XML, KDL, HCL, NestedText, INI — and works out where each is the
better choice. It is written from the OGDL side, but tries to be fair: OGDL is
the right tool for a narrower set of jobs than its proponents would like, and
the sections below say which.

Statements about OGDL refer to the specifications in [`rfc/`](../rfc/) and to
this implementation. Statements about other languages refer to their current
specifications (JSON: RFC 8259; YAML 1.2.2; TOML 1.0.0; XML 1.0; KDL 2.0),
with notes where common implementations depart from them.

---

## 1. The decisive difference: the data model

Most practical differences between data languages follow from their data model,
not their syntax. Syntax is a matter of taste and tooling; the data model
decides what can be said at all.

| Language | Structure | Order of named entries | Repeated names | Scalar types |
|---|---|---|---|---|
| **OGDL** | ordered tree of named nodes | significant | allowed | none — all text |
| **JSON** | objects (maps) and arrays | not significant | "SHOULD be unique"; behaviour undefined | string, number, bool, null |
| **YAML** | maps, sequences, scalars (a graph, via anchors) | not significant | forbidden by spec; many parsers accept silently | resolved by schema: str, int, float, bool, null, … |
| **TOML** | tables (maps) and arrays | not significant | forbidden, an error | string, int, float, bool, 4 date/time kinds |
| **XML** | ordered tree of elements, plus attributes and text | significant for elements; not for attributes | allowed for elements | none without a schema |
| **KDL** | ordered nodes with arguments, properties, children | significant | allowed | string, number, bool, null; type annotations |
| **HCL** | blocks, attributes, expressions | blocks ordered; attributes not | blocks yes, attributes no | string, number, bool, null, collections |
| **NestedText** | dicts, lists, strings | dicts are ordered in practice; keys unique | no | none — all text |
| **INI** | sections of key/value pairs | unspecified (no standard) | unspecified | none |

Two groups emerge.

**Map-and-list languages** (JSON, YAML, TOML, HCL attributes, NestedText):
every container is either a *map* — named, unordered, unique keys — or a
*list* — ordered, unnamed. To express "an ordered sequence of named things",
you must choose one and encode the other: a list of single-key maps, a map
plus a separate ordering field, or an array of tables. The encoding is a
convention, and every consumer must know it.

**Ordered-tree languages** (OGDL, XML, KDL): one container type, in which
every child is both named and positioned. Order and repetition cost nothing,
because they are not a special case.

OGDL is the most minimal member of the second group. A node is a name and a
list of children — nothing else. XML adds attributes, text content and
namespaces; KDL adds positional arguments, properties and typed values. OGDL
has none of these: `port 8080` is two nodes, `port` with child `8080`, and
whether that is a "key with a value" is for the reader to decide (RFC §2.2).

OGDL's line syntax looks like KDL's — `rule allow any` reads as a name with
arguments — but it is notation only: the arguments are leaf children, with an
equivalent multi-line form (RFC §6.5). KDL's properties (`class=asil`) have no
OGDL counterpart; `=` is plain text. The model keeps its single container type;
the syntax just lets a node and its leaf children share a line.

### What the minimal model buys

- **Order is data.** A rule list evaluated first-match, a sequence of build
  steps, labels whose rank derives from declaration order — all are
  represented directly, and survive any conforming parser. In JSON or YAML,
  order inside a map survives only by implementation accident.
- **Repetition is data.** Several `role` or `include` or `route` entries under
  one parent need no wrapping array and no invented plural key.
- **No type guessing.** `no`, `0755`, `1e3`, `22:22`, `NULL` and `2026-10-05`
  all stay text. The YAML 1.1 class of surprise — `country: no` becoming
  `false`, `22:22` becoming `1342` — cannot happen, because the stream carries
  no types for a parser to infer.
- **Canonical form.** Two OGDL documents denote the same tree if and only if
  their canonical serializations are byte-identical (RFC §8.3). Equality,
  hashing, signing, caching and meaningful diffs need no extra standard. JSON
  needs a separate one for this (RFC 8785, JCS); YAML and TOML have none.
- **A small attack surface.** No anchors or aliases (YAML's "billion laughs"),
  no entity expansion (XML's), no includes, no object construction tags
  (unsafe YAML loaders), no implicit coercion (RFC §12).

### What it costs

- **Every consumer must interpret.** "Port is an integer", "tags is a list",
  "debug is a boolean" live in the application, not the document. A JSON
  consumer gets `8080` as a number for free; an OGDL consumer calls `Int64()`
  and decides what to do if it fails. The schema language
  ([`ogdl-schema-rfc.md`](../rfc/ogdl-schema-rfc.md)) gathers these decisions
  into one document, which can also convert the values, but it is young and
  exists only in this implementation.
- **No null, no empty-versus-missing convention.** A leaf `port` with no child,
  a `port ""`, and an absent `port` are three distinct trees, but what they
  *mean* is again the application's call.
- **A record does not fit on one line.** The names after the first on a line
  are leaf children, so `S1 value 1.0 class asil` is `S1` with four flat
  children; a record with fields takes a line per field. The mistake, when
  made, is shallow and visible in any dump. (Earlier drafts read a line as a
  chain, which buried the error several levels deep; see
  [`ogdl-note.md`](../ogdl-note.md). A `key=value` marker was tried and
  withdrawn; see `rfc/analysis.md` Part 5.)
- **Mapping to program types is manual.** A JSON object maps onto a struct or
  dictionary with one library call in every mainstream language. OGDL has no
  standard mapping, because repeated names and order do not fit a map.

---

## 2. The same data in each language

A server configuration with an ordered rule list, evaluated first-match.

**OGDL**

```
server
  host db.example.com
  port 8080
  # first match wins
  rule allow 10.0.0.0/8
  rule allow 192.168.0.0/16
  rule deny any
```

**JSON**

```json
{
  "server": {
    "host": "db.example.com",
    "port": 8080,
    "rules": [
      { "action": "allow", "from": "10.0.0.0/8" },
      { "action": "allow", "from": "192.168.0.0/16" },
      { "action": "deny",  "from": "any" }
    ]
  }
}
```

**YAML**

```yaml
server:
  host: db.example.com
  port: 8080
  # first match wins
  rules:
    - { action: allow, from: 10.0.0.0/8 }
    - { action: allow, from: 192.168.0.0/16 }
    - { action: deny,  from: any }
```

**TOML**

```toml
[server]
host = "db.example.com"
port = 8080

# first match wins
[[server.rules]]
action = "allow"
from = "10.0.0.0/8"

[[server.rules]]
action = "allow"
from = "192.168.0.0/16"

[[server.rules]]
action = "deny"
from = "any"
```

**XML**

```xml
<server>
  <host>db.example.com</host>
  <port>8080</port>
  <!-- first match wins -->
  <rule action="allow" from="10.0.0.0/8"/>
  <rule action="allow" from="192.168.0.0/16"/>
  <rule action="deny" from="any"/>
</server>
```

**KDL**

```kdl
server {
    host "db.example.com"
    port 8080
    // first match wins
    rule "allow" "10.0.0.0/8"
    rule "allow" "192.168.0.0/16"
    rule "deny" "any"
}
```

Observations:

- The map-and-list languages each had to invent a plural container (`rules`)
  and field names (`action`, `from`) that the OGDL, XML and KDL versions do
  without, because a repeated `rule` is already an ordered list.
- JSON alone cannot carry the comment.
- TOML's array-of-tables syntax keeps each rule readable but scatters a single
  logical list across the file; deeper nesting makes this worse
  (`[[server.listeners.rules]]`).
- KDL is the closest relative of OGDL: same model, with braces instead of
  indentation and typed values instead of plain text. `8080` there is a number;
  in OGDL it is text. The `rule` lines are even the same shape: a name followed
  by its arguments.
- Where field names help, OGDL can name them at the cost of a line each, and
  the result is still the same ordered list:

  ```ogdl
  rule
    action allow
    from 10.0.0.0/8
  ```

---

## 3. Feature comparison

| | OGDL | JSON | YAML | TOML | XML | KDL |
|---|---|---|---|---|---|---|
| Comments | yes | no (JSONC/JSON5 add them) | yes | yes | yes | yes |
| Comments kept when a program rewrites a file | yes (`ParseDocument`) | — | some libraries (ruamel.yaml) | some libraries (tomlkit) | yes, in DOM parsers | some libraries |
| Structure by | indentation | brackets | indentation (or brackets) | headers + `=` | tags | braces |
| Multi-line text | blocks, multi-line quotes | escapes only | several block styles | `"""` strings | text content, CDATA | multi-line strings |
| Typed scalars | no | yes | yes, schema-dependent | yes, incl. dates | no (schema adds) | yes, plus annotations |
| Null | no | yes | yes | no | no | yes |
| References / anchors | no | no | yes | no | entities, IDs | no |
| Canonical form | yes, in the core spec | separate standard (RFC 8785) | no | no | separate (C14N) | no |
| Schema languages | OGDL Schema (this implementation) | JSON Schema | JSON Schema (via data model) | community schemas | XSD, RELAX NG, DTD | KDL Schema (draft) |
| Path / query | OGDL Path | JSONPath (RFC 9535), JMESPath | via JSON tools | via JSON tools | XPath, XQuery | KDL Query |
| Templating built in | yes (this implementation) | no | no | no | XSLT | no |
| Specification size | small | very small | very large | small | large (plus family) | small |
| Ecosystem | very small | universal | very large | large | very large, ageing | small, growing |

Notes:

- **Comment preservation** is a property of the parser, not the language.
  OGDL's comments are not data (RFC §2.5): they create no nodes and do not
  affect equality. A parser may still keep them beside the tree, and this one
  does: `ParseDocument` attaches each comment to a node (RFC Appendix B), so a
  program can change the tree and write the file back with its comments, blank
  lines and inline layout intact. YAML and TOML do the same through particular
  libraries (ruamel.yaml, tomlkit), which matters for tools that edit
  human-maintained configuration.
- **Indentation errors.** OGDL rejects a mix of tabs and spaces, and a dedent
  to a level that was never opened. Python does the same (`TabError`,
  `IndentationError: unindent does not match any outer indentation level`),
  and YAML forbids tabs for indentation outright. This is strictness, with an
  error at a line and column, not a trap: the alternative is guessing a tab
  width, and getting a different tree from what the author sees.
- **Spec size** is not trivia. YAML 1.2's specification is long, and real
  parsers disagree on edge cases; many still default to YAML 1.1 type
  resolution. JSON's tiny grammar is why every language parses it the same way.
  OGDL's core RFC is short and its Appendix A test vectors pin the hard cases,
  but with few implementations this has not yet been tested by divergence.

---

## 4. Fitness by task

### 4.1 Configuration written by people

What matters: readability, comments, low punctuation, error messages that point
at the mistake, and no surprises.

| Choice | When |
|---|---|
| **TOML** | Flat or shallow configuration: tool settings, package manifests (`Cargo.toml`, `pyproject.toml`). Typed values, unambiguous, comment-friendly. Becomes awkward past two or three levels of nesting. |
| **YAML** | When the ecosystem already chose it (Kubernetes, CI pipelines, Ansible). Expressive, but implicit typing, indentation subtleties and a large spec cause real incidents. Prefer quoted strings and a 1.2 parser. |
| **OGDL** | Deep or ordered configuration — rule sets, pipelines, hierarchies where position matters or names repeat — written by people. One structural rule at every depth (a line is a node and its leaf children), comments that survive programmatic edits, no type surprises, strict errors with line and column. |
| **KDL** | The same niche as OGDL, for users who want typed values and explicit braces, and a slightly larger ecosystem. |
| **HCL** | Infrastructure configuration that needs expressions, variables and functions inside the file (Terraform). |
| **JSON** | Avoid for hand-written configuration: no comments, trailing-comma errors, quoting noise. JSONC or JSON5 relieve some of this. |
| **INI** | Trivial key/value settings; no standard, so behaviour varies per parser. |

OGDL's strongest case is configuration whose meaning depends on order — the
`first_match` rule list, the ranked label set — where map-based languages force
an encoding the author must remember and the reader must decode. Its weakest
case is configuration that is essentially a flat set of typed settings, which
TOML expresses with less ceremony and more checking.

### 4.2 Data interchange between programs

What matters: universal parser availability, a precise mapping to program
types, performance, and interoperability across languages.

| Choice | When |
|---|---|
| **JSON** | The default for APIs and messages. Parsers everywhere, a direct mapping to native types, a schema language, query languages. Watch large integers (beyond 2^53 they lose precision in JavaScript) and duplicate keys. |
| **Protobuf, CBOR, MessagePack** | Volume or latency matters, or a schema should be enforced at the boundary. Binary, compact, not human-readable. |
| **XML** | Document-centric interchange, mixed content, established industry schemas (office formats, SOAP, many government and financial standards). |
| **CSV** | Tabular data. |
| **OGDL** | Rarely the right choice between independent parties today: the partner would need an OGDL parser, and there are few outside Go. Reasonable inside one system, where both ends use this package and order or repetition must survive the trip. The canonical form is a genuine asset when messages are hashed, signed or compared. |

### 4.3 Documents, logs and streams of records

- **Structured text documents** with markup inside prose are XML's territory;
  none of the others handles mixed content.
- **Line-oriented logs** favour JSON Lines: one record per line, appendable,
  greppable. OGDL's canonical form is also one node per line and appendable,
  but a record spans several lines, so line tools see fragments.
- **Human-readable dumps and debug output** suit OGDL well: canonical
  serialization is stable, indented and quote-light, so dumps diff cleanly.

### 4.4 Data with computation: templates, expressions, generated text

Two designs exist.

1. **Computation inside the data language** — HCL, Jsonnet, CUE, Dhall,
   Nickel. Configuration files contain variables, functions and conditionals.
   Powerful for large configuration estates; the price is that reading a
   value may mean running a program, and that the language must be learned as
   a language.
2. **Data stays inert; computation is a separate layer** — OGDL here, with its
   path, expression and template RFCs; also XML with XPath/XSLT, and JSON with
   external templating (Jinja, Go templates).

OGDL takes the second route deliberately: the core RFC has no executable
constructs, and the expression and template layers operate *on* a tree rather
than inside a document. That keeps untrusted documents safe to parse, while
still giving the host program `$for(x, cart)` and `user.age >= 18` over the
same tree with the same path syntax. Note that evaluating an expression *can*
modify the tree (assignment), so expressions from untrusted sources should run
against a copy.

Compared with Go's `text/template` over JSON-decoded data, OGDL templates
address the tree with the same path language used everywhere else, report
errors instead of swallowing them, and do not require decoding into Go types
first. Compared with XSLT, they are far smaller and far less capable.

### 4.5 Knowledge representation and ordered hierarchies

Taxonomies, bills of materials, outlines, risk scales, menu trees — data that
*is* a tree, where siblings have an order and may share names. Here the
ordered-tree languages are the natural fit and OGDL is the most direct: the
document is the tree, with no element/attribute decision (XML) and no
argument/property/child decision (KDL) to make for each item.

---

## 5. Common pitfalls, by language

| Language | Pitfall |
|---|---|
| OGDL | A one-line record needs `=`: `S1 value 1.0` is `S1` with two children, not a field and its value. All values are text, so every consumer converts. Few implementations outside Go. |
| JSON | No comments. Large integers lose precision in JavaScript. Duplicate keys are accepted silently by most parsers, with differing results. Object order may or may not be preserved. |
| YAML | Implicit typing under 1.1 rules (`no`, `on`, `0755`, `22:22`). Unsafe loaders that construct arbitrary objects. Alias expansion bombs. A large spec that parsers implement differently. |
| TOML | Deep nesting is verbose; a logical list of records is spread across `[[...]]` headers. No null. Inline tables cannot span lines in 1.0. |
| XML | Verbosity. The element-or-attribute choice has no right answer. External entity (XXE) and entity expansion attacks with default parser settings. Namespaces. |
| KDL | Version 1 and 2 differ in syntax (bare strings, booleans as `#true`). Small ecosystem. |
| HCL | Effectively tied to HashiCorp tools; the expression language makes files programs. |

---

## 6. Decision guide

- **You need machines from different organisations to exchange data** → JSON
  (or a binary format with a schema). Ecosystem beats elegance.
- **People edit flat or shallow settings** → TOML.
- **The tool you configure already mandates something** → use that; fighting
  an ecosystem costs more than any syntax gains.
- **The data is a tree whose order matters or whose names repeat**, and it is
  written or read by people → OGDL or KDL. Choose OGDL for the most minimal
  syntax, no type guessing, canonical equality, and an integrated path,
  expression and template layer in Go; choose KDL for typed values, braces and
  wider language support.
- **The data is a document with prose and markup** → XML.
- **Configuration needs variables, functions and reuse across many files** →
  CUE, Jsonnet, Dhall or HCL; or OGDL plus its template layer, if keeping data
  inert matters more than having computation in the file.

## 7. Summary

OGDL's case rests on one idea: an ordered list of named nodes at every level
is a better default container than the map/list pair, because it represents
order and repetition without encoding. From that follow its real advantages —
no type guessing, canonical equality, a minimal and safe core, and a uniform
path, expression and template layer on one tree.

Its disadvantages are equally real: almost no ecosystem, a schema language
with a single implementation, and no standard mapping to program types. These make it a poor choice for open interchange and for flat,
typed settings, and a strong one for ordered, hierarchical, human-maintained
data processed by programs that use this package.

## References

- OGDL: [`rfc/ogdl-rfc.md`](../rfc/ogdl-rfc.md),
  [`rfc/ogdl-path-rfc.md`](../rfc/ogdl-path-rfc.md),
  [`rfc/ogdl-expr-rfc.md`](../rfc/ogdl-expr-rfc.md),
  [`rfc/ogdl-template-rfc.md`](../rfc/ogdl-template-rfc.md),
  [`ogdl-note.md`](../ogdl-note.md)
- JSON: RFC 8259; I-JSON, RFC 7493; JSON Canonicalization Scheme, RFC 8785;
  JSONPath, RFC 9535
- YAML: YAML Ain't Markup Language, version 1.2.2 (yaml.org/spec)
- TOML: TOML v1.0.0 (toml.io)
- XML: Extensible Markup Language 1.0 (W3C); Canonical XML (W3C)
- KDL: kdl.dev
- HCL: github.com/hashicorp/hcl
- NestedText: nestedtext.org
