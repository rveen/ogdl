# OGDL Path: audit of https://ogdl.org/spec/path.html

> **Status: superseded by `ogdl-path-rfc.md`.** Every defect below is resolved
> there; this document is kept as the rationale record. Section 2 describes the
> semantics as first implemented — where it differs from the RFC (token class,
> quoted escapes, selector placement), the RFC governs.

Findings from checking the published path specification against the reference
implementation (`github.com/rveen/ogdl`), before implementing `Get` here.
Behaviour marked *verified* was observed by running the reference package.

The published grammar is:

```
[1] path          ::= element ( '.' element )*
[2] basic_element ::= token | quoted | index | selector
[3] element       = basic_element (index|selector)?
[4] index         ::= '[' number ']'
[5] selector      ::= '{' number? '}'
[6] token         := (unicode_letter | unicode_digit | '_')+
[7] quoted        := '\'' string '\'' | '"' string '"'
[8] string        := (unicode_letter | unicode_mark | unicode_number |
                      unicode_punctuation | unicode_symbol | unicode_separator_space )+
[9] number        := (unicode_digit)+
```

## 1. Defects in the specification

### 1.1 The result of evaluating a path is never defined

The document says what each element *selects* but never says what a path
*returns* — one node, the text below a node, or a list. Every other ambiguity
below follows from this one. The examples show both singular results
(`chapter.title`) and plural ones (`chapter.p{}`) without saying how a caller
distinguishes them.

This is the first thing a path RFC has to fix: state the result type, and state
it uniformly for all three element kinds.

### 1.2 Index and selector are mutually inconsistent (reference implementation)

`eval.go` builds a synthetic wrapper node for both, but fills it differently:

```go
r.Add(ctx.GetAt(ix))       // index: the wrapper holds the selected node
r.AddNodes(nn)             // selector: the wrapper holds the selected node's children
```

*Verified* consequence — a path can be continued after `{n}` but not after `[n]`:

| Path | Reference result |
|---|---|
| `book.chapter{1}.title` | `Second` |
| `book.chapter[1].title` | **nil** |

Nothing in the specification licenses this difference; it reads as an
implementation accident. Either both wrap the nodes or both wrap their children.

### 1.3 Out-of-range index produces a bogus node (reference implementation)

*Verified*: `list[9]` on a three-element list returns a node of length 1 whose
content is a typed-nil `*Graph`, not an empty result. `GetAt` returns a typed nil
which `Add` then fails to recognize as nil — the classic Go typed-nil-in-
interface trap. Out-of-range indexing should yield nothing.

### 1.4 The grammar is ambiguous about chains

Production [2] makes `index` and `selector` basic elements, and production [3]
allows one more index or selector after a basic element. So `[0][1]`, `a{0}{1}`
and `[0]{1}` are all derivable, while the semantics of a selector that follows
an index — or of a second selector — are undefined. A selector counts
occurrences of "the token written before" it, and after an index there is no
such token.

The reference rejects that case at evaluation time (`ErrInvalidIndex` when the
previous element is empty), which means the grammar accepts strings the
evaluator refuses. The grammar should be restricted so that a selector can only
follow a token or quoted element.

### 1.5 A selector cannot be the first element

Production [1] permits `{0}` as a whole path, but a leading selector has no
preceding token to count. Same class of problem as 1.4.

### 1.6 The token class is too narrow, and the escape hatch is not stated

Production [6] admits only letters, digits and `_`. OGDL *names*, by contrast,
may hold any character except space, break and control (RFC 4.3). So a great
many perfectly ordinary nodes cannot be addressed by an unquoted token:

```
ip
  192.168.0.10        -> needs 'quoting': dots are separators
eth0-1                -> needs quoting: '-' is not in the token class
```

Quoting is the escape hatch, but the specification never says so, and a reader
comparing [6] with the OGDL word class will reasonably conclude that such names
are unreachable. Either widen [6] to "any character that is not `.`, `[`, `]`,
`{`, `}`, a quote or a space", or state explicitly that anything outside [6]
must be quoted.

### 1.7 Quoted elements have no escape mechanism

Production [8] admits any graphic character and inline space, so a quoted
element cannot contain its own delimiter. Using the other delimiter covers one
case, but a name containing **both** `'` and `"` is unaddressable. OGDL itself
solves this with `\'` and `\"` (RFC 4.4.1); path quoting should do the same.

### 1.8 Wording

"Indexes ( [n] ) select the nth-1 subnode" — this means zero-based, but "nth-1"
invites off-by-one readings. Say "the index is zero-based: `[0]` selects the
first subnode".

The text should also say explicitly *what* `[n]` indexes: the children of the
current node. `chapter[1]` is the second child of the first chapter, **not** the
second chapter — that is `chapter{1}`. This is the single most likely misreading
of the whole document.

### 1.9 `{}` is ambiguous between two readings

"An empty selector ( {} ) selects all subnodes with the specified node name."
The reference instead accumulates the *children* of all such subnodes
(`addEqualNodes` + `AddNodes`). *Verified*: `book.chapter{}` returns five nodes —
the merged children of both chapters — not the two chapters. The spec's own
example, `chapter.p{}`, cannot distinguish the readings, since each `p` has
exactly one child.

### 1.10 The specification is incomplete relative to the language in use

The reference parses argument lists and groups — `a(b)`, `a.(b)`, `(a, b)` —
which have no production in the published grammar. `NewPath` documents them as
"extended paths, as those used in templates". Production [9] also restricts an
index to digits, while the implementation parses a whole expression there, so
`[-1]` and `[i]` are accepted by the code and excluded by the grammar.

A path RFC should either specify these or state that they belong to a separate
expression language layered on top.

**Resolved.** Both were done: `ogdl-path-rfc.md` states that they belong to a
separate layer, and `ogdl-expr-rfc.md` specifies that layer — calls, groups and
expression indexes included.

## 2. Semantics implemented here

Given 1.1, this package had to choose. The rules are uniform, and are the
minimum needed to make the specification's own examples work:

* **A path evaluates to a node**, never to nil. A path that matches nothing, or
  that is malformed, yields an empty node, so calls can be chained.
* **A token** selects the first child whose *textual form* equals it. The result
  is that child, so `String()` reads the text below it and `ThisString()` its own
  text. This preserves the canonical example, `g.Get("eth0.ip").String()`.
* **`[n]`** selects the n-th child of the current node, zero-based, and returns
  **that node itself** — the same convention as a token, which is what 1.2 gets
  wrong. Read it with `ThisString()`; out of range yields an empty node (1.3).
* **`{n}`** selects the n-th child, among those matching the preceding token,
  of the node that token was matched in. It returns that node, so a path can
  continue through it.
* **`{}`** selects *all* such nodes — the reading of the specification text, not
  of the reference implementation (1.9) — and returns a **set**: a synthetic node
  whose children are the matches. Iterate `Out`.
* **A token after a set** maps over its members, taking one match from each, so
  `chapter{}.title` gives every chapter's title. This is what makes the
  specification's `chapter{}.title{}` example work without the trailing `{}`.
* A selector must follow a token or quoted element (1.4, 1.5); otherwise the
  path is rejected as malformed.
* The token class is exactly production [6], and quoting is the documented escape
  hatch (1.6). Path quoting does not yet support escapes (1.7).
* Argument lists and groups (1.10) are not implemented.

## 3. Differences from the reference implementation

Deliberate, and a consequence of section 2:

| Path | Reference | Here |
|---|---|---|
| `book.chapter[1].title` | nil | the title node of chapter 1's second child, if any |
| `list[0]` | `String()` = `a` | `ThisString()` = `a`, `String()` = `""` |
| `list[9]` | node of length 1 holding a typed nil | empty node |
| `book.chapter{}` | 5 nodes (merged children) | 2 nodes (the chapters) |
| `book.chapter{}.title` | `First` only | both titles |
| malformed path | silently empty | `GetErr` reports a `*PathError` |

The `list[0]` row is the one to watch when porting: an index selects an item, so
its text is read with `ThisString()`, not `String()`.
