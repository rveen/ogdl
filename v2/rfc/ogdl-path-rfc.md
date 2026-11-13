# RFC: OGDL Path

## Status of This Memo

This document specifies OGDL Path, an addressing notation for the Ordered Graph Data Language (OGDL), as an informational specification in the style of an IETF RFC.

It revises the earlier description at <https://ogdl.org/spec/path.html>. That document left the result of evaluating a path undefined, and its grammar derived strings its own evaluation rules could not interpret. The defects addressed here, and the reasoning behind each change, are recorded in `path-notes.md`.

This document depends on `ogdl-rfc.md` ("the core specification"), which defines the OGDL data model and stream syntax.

Distribution of this memo is unlimited.

The key words MUST, MUST NOT, REQUIRED, SHALL, SHALL NOT, SHOULD, SHOULD NOT, RECOMMENDED, MAY, and OPTIONAL in this document are to be interpreted as described in RFC 2119.



## Abstract

OGDL Path is a compact textual notation for addressing nodes within an OGDL tree. A path is a sequence of elements: names to descend by, positional indexes, and selectors that choose among repeated names. Evaluating a path against a tree yields a sequence of nodes. This document defines the lexical rules, grammar, evaluation semantics, error handling, and the use of a path as a fragment identifier for the `text/ogdl` media type.



## 1. Introduction

An OGDL document is an ordered tree in which sibling names may repeat (core specification, section 2.4). Addressing a node therefore requires more than a dotted name: it requires a way to say *which* of several same-named siblings is meant, and a way to say *all of them*.

OGDL Path provides three element kinds for this:

```
book.chapter.title      the title of the first chapter
book.chapter{1}.title   the title of the second chapter
book.chapter{}.title    the title of every chapter
list[2]                 the third child of list
```

A path is a standalone character string. It is not part of an OGDL stream and never appears inside one; the two languages share no lexical rules.

### 1.1 Relationship to the Earlier Specification

Any path that was valid and unambiguous under <https://ogdl.org/spec/path.html> remains valid here and selects the same nodes, with one exception:

* A selector must immediately follow a name element (section 5). Strings such as `{0}` or `a[0]{1}`, which the earlier grammar derived but whose meaning it never defined, are now syntax errors.

The token class is unchanged and remains narrow (section 4.2). What this document adds is the explicit requirement to quote anything outside it, and escape sequences within quoted elements (section 4.3), so that every legal OGDL name is addressable. Both additions widen what can be expressed without invalidating any existing path.



## 2. Evaluation Model

### 2.1 Result

Evaluating a path against a node yields an ordered **node sequence**: zero or more nodes of the tree, in document order, possibly with repetitions.

A path that matches nothing yields the empty sequence. This is not an error (section 8).

The nodes in the result are the nodes of the tree itself, not copies. This specification defines no synthetic or wrapper nodes; an implementation MUST NOT introduce a node that is not in the tree into the result.

### 2.2 Presenting the Result

An API MAY present a result of length one as a bare node, and a result of any other length as a list or as a synthetic container. That choice is outside this specification, but an implementation SHOULD document it, since it determines how a caller distinguishes "one match" from "several".

### 2.3 Name Comparison

A path element matches a node by comparing the element's text with the node's **name**: the textual form the node takes in a stream (core specification, sections 2.4 and 8.2).

Comparison is character-by-character over Unicode characters. No normalization of any kind — case, Unicode NFC/NFD, whitespace — is performed. A node whose in-memory representation is not textual is compared by the text that represents it.



## 3. Character Model

A path is a sequence of Unicode characters, decoded before parsing, exactly as for a stream (core specification, section 3.1).

The following characters are **reserved** and have syntactic meaning in a path:

| Character | Role |
|---|---|
| `.` U+002E | element separator |
| `[` `]` U+005B, U+005D | index |
| `{` `}` U+007B, U+007D | selector |
| `'` `"` U+0027, U+0022 | quoted element delimiters |
| `(` `)` `,` U+0028, U+0029, U+002C | call and group elements, in an expression context only (section 5.1) |

Control characters (Unicode General Category Cc) and U+0020 SPACE MUST NOT appear in a path except within a quoted element, where SPACE is permitted.

Every character outside this table and outside the token class of section 4.2 — every operator, sign and punctuation mark — is undefined in a path and MUST be rejected outside a quoted element. Those characters belong to the expression layer (section 10), which never lexes them as part of a path.



## 4. Lexical Elements

### 4.1 Number

```
number = DIGIT+          ; DIGIT is U+0030..U+0039
```

Numbers are decimal, zero-based, and non-negative. Only ASCII digits are permitted: other Unicode decimal digits are not numbers and MUST NOT be accepted, so that parsing is independent of the Unicode version in use.

Leading zeros are permitted and are not significant. Implementations MUST document the largest index they accept (section 9).

### 4.2 Token

```
token      = token_char+
token_char = unicode_letter | unicode_digit | "_"
```

This is the narrow class of the earlier specification, retained deliberately. Every other character is left free for the expression layer (section 10): operators such as `+`, `-`, `*`, `/`, `!`, `<` and `=` must not be absorbable into a token, or `a-b` would be one name in a path and a subtraction in an expression, and the two languages could not share a parser.

The cost is that most punctuation in a node's name puts it out of reach of an unquoted token. **Any name that is not composed solely of letters, digits and underscore MUST be written as a quoted element** (section 4.3). The earlier specification left this unsaid, which is what made the narrow class look like an oversight rather than a choice; it is stated here as a requirement:

```
host.'192.168.0.10'      ; dots
iface.'eth0-1'           ; hyphen
config.'my setting'      ; space
op.'a+b'                 ; operator characters
```

A conforming implementation MUST NOT accept a wider unquoted token class as an extension. Doing so would make paths that are valid locally fail elsewhere, and would collide with the expression layer.
```

### 4.3 Quoted Element

```
quoted      = ( "'" qchar('")* "'" ) | ( '"' qchar('"')* '"' )
qchar(q)    = ( any character except q, "\", or a control character ) | escape
escape      = "\" ( "\" | "'" | '"' )
```

A quoted element denotes exactly the characters it contains, after escape processing. The delimiters are not part of the name, and the choice of delimiter carries no meaning.

The escape sequences exist so that a name containing both quote characters is addressable; the earlier specification had no escape mechanism and could not express such a name. Escapes are a strict subset of those defined for streams (core specification, section 4.4.1); `\n` and `\t` are **not** defined here, because a name containing a break or a tab is addressed by writing that character's stream escape in the node, not in the path.

Any other sequence beginning with `\` is an error, reserving the space for future extension.

An unterminated quoted element is an error.

### 4.4 Index and Selector

```
index    = "[" number "]"
selector = "{" number? "}"
```

An index with no number, such as `[]`, is not defined and is an error. A selector with no number, `{}`, is the **empty selector** and has the distinct meaning given in section 6.4.

In an expression context, an index may contain a whole expression rather than a number; see the expression RFC, section 5.1. The meaning of `a[0]` is unchanged — only what may appear between the brackets is widened. A selector always takes a number.



## 5. Grammar

```
path        = head tail*
head        = name_element | index
tail        = ( "." name_element ) | index | selector
name_element = token | quoted
```

with one constraint that the productions above do not express:

* A `selector` MUST be immediately preceded by a `name_element`.

A path that violates it is malformed. This is what makes `{0}`, `a[0]{1}` and `a{}{0}` errors: a selector counts occurrences of a name, and in each of those there is no name for it to count.

Notes:

* The separator `.` is REQUIRED between two name elements and MUST NOT appear elsewhere. It is optional before an index or a selector, so `a[0]` and `a.[0]` are the same path, as are `a{1}` and `a.{1}`.
* A path is never empty; the empty string is not a path.
* A trailing `.` is malformed.

### 5.1 Extended Paths

Within an expression, `tail` admits two further elements, defined by the expression RFC section 5:

```
tail =/ call | group
call  = "(" arg_list? ")"
group = "." "(" expression ")"
```

These exist only in an expression context. An implementation MUST NOT accept them where a path of this document is expected: in a strict path, `(` and `)` are errors. The two languages therefore remain distinguishable, and an implementation is expected to offer both — a strict parser for its addressing API, and an extended one for its evaluator.



## 6. Evaluation (Normative)

### 6.1 State

Evaluation proceeds left to right over the elements of the path, carrying:

* `S` — the current node sequence, initially the single node the path is evaluated against.
* `P` — the sequence `S` held when the most recent name element was evaluated.
* `t` — the text of the most recent name element.

`P` and `t` are undefined until a name element has been evaluated. By the constraint in section 5, a selector cannot be reached while they are undefined.

Every element rewrites `S`. If `S` becomes empty, evaluation stops and the result is the empty sequence.

### 6.2 Name Element

Set `P` to `S` and `t` to the element's text. Then:

> For each node `n` of `P`, in order, append to the new `S` the **first** child of `n` whose name equals `t`, if any.

A name element therefore never widens the sequence. Applied to a single node it yields at most one node; applied to a sequence produced by an empty selector, it maps over that sequence.

### 6.3 Index

`[k]` rewrites `S` as:

> For each node `n` of `S`, in order, append to the new `S` the child of `n` at position `k`, counting from zero, if `n` has one.

An index out of range contributes nothing. It is not an error and MUST NOT contribute a placeholder or null node.

An index selects a child **by position among all children**, not among children matching a name. `chapter[1]` is the second child of the first chapter; the second chapter is `chapter{1}`.

`t` becomes undefined after an index, which is why a selector may not follow one.

### 6.4 Selector

A selector reconsiders the children that the immediately preceding name element chose among.

`{k}` rewrites `S` as:

> For each node `p` of `P`, in order, append to the new `S` the child of `p` at position `k`, counting from zero, among those children of `p` whose name equals `t`, if there is one.

The empty selector `{}` rewrites `S` as:

> For each node `p` of `P`, in order, append to the new `S` **every** child of `p` whose name equals `t`.

`{0}` therefore selects exactly what the name element alone selects: `a` and `a{0}` are equivalent.

The empty selector is the only element that can lengthen the sequence.

Note that `{}` selects the matching nodes themselves. It does not select their children, and it does not merge them. `book.chapter{}` is the sequence of chapters, so `book.chapter{}.title` is the title of each chapter (section 6.2).

### 6.5 Worked Example

Given:

```
book
  chapter
    title First
    p one
    p two
  chapter
    title Second
    p three
```

| Path | Result |
|---|---|
| `book.chapter.title` | the `title` node of the first chapter |
| `book.chapter.title.String()` (informative) | `First` |
| `book.chapter{1}.title` | the `title` node of the second chapter |
| `book.chapter{}` | both `chapter` nodes |
| `book.chapter{}.title` | both `title` nodes |
| `book.chapter{}.p{}` | all three `p` nodes |
| `book.chapter.p{}` | the two `p` nodes of the first chapter |
| `book.chapter[0]` | the first chapter's first child, the `title` node |
| `book.chapter[9]` | the empty sequence |
| `library.chapter` | the empty sequence |



## 7. Fragment Identifier

A path MAY be used as the fragment identifier of a `text/ogdl` resource, which the core specification (section 13.1) reserves for this purpose.

The fragment identifies the node sequence obtained by evaluating the path against the root node of the resource. Characters not permitted in a fragment by RFC 3986 MUST be percent-encoded; percent-decoding happens before path parsing.

An empty sequence identifies nothing; it is not an error, and a consumer SHOULD treat it as it treats an unmatched fragment in other media types.



## 8. Error Handling

A conforming implementation MUST reject the following as malformed paths:

* The empty string.
* A separator `.` that is not between two name elements, including a trailing `.`.
* A selector not immediately preceded by a name element (section 5).
* An index with no number, a non-terminated index or selector, or a number containing a non-ASCII digit.
* An unterminated quoted element, or one containing a control character.
* An undefined escape sequence in a quoted element (section 4.3).
* Any reserved character in a position where this grammar does not define it.
* An index exceeding a documented implementation limit (section 9).

An error report SHOULD identify the character offset within the path.

A path that is well-formed but matches nothing is **not** an error (section 2.1). Implementations MUST distinguish the two conditions: a malformed path is a fault in the caller, an empty result is a fact about the data.



## 9. Security Considerations

A path is a query, not a program. It defines no iteration, no recursion and no computation, so evaluation terminates on every input.

Implementations processing untrusted paths SHOULD:

* Impose and document a maximum path length and a maximum index value, so that a long digit run cannot overflow an integer.
* Bound the result size. The empty selector is the only element that widens a sequence, but a chain of them multiplies: `a{}.b{}.c{}` can select a number of nodes proportional to the product of the branching factors. Evaluation is otherwise linear in the size of the tree traversed.
* Not treat path evaluation as an access control boundary. A path can address any node in the tree it is evaluated against; restricting access means evaluating against a subtree, not filtering paths.



## 10. Extensibility

A path is the addressing half of a larger notation whose other half is an expression language. That half is now specified: see `ogdl-expr-rfc.md`, which defines argument lists such as `a(b)`, groups such as `a.(b)`, expressions in place of a number inside `[]`, and arithmetic, comparison and boolean operators.

The token class of section 4.2 is defined by **inclusion** — letters, digits and underscore, and nothing else. That is what made the extension possible without breaking anything. Because no other character can be part of an unquoted token, the expression layer was able to claim every operator character without changing the interpretation of a single conforming path. Had the token class been defined by exclusion, `a-b` would already have been one token, and introducing subtraction would have silently reinterpreted existing paths.

The extension point is now **spent**. The operator set of the expression RFC section 3.3 is closed, and its section 11 reserves what little remains — a bare `=`, `&` or `|`, for assignment. A conforming implementation that accepts unquoted operator characters in path tokens breaks the expression layer, and MUST NOT do so.



## 11. Conformance

A conforming implementation:

* MUST evaluate every well-formed path as defined in section 6, yielding the node sequence specified there.
* MUST reject every malformed path (section 8), and MUST NOT silently treat one as an empty result.
* MUST NOT introduce nodes that are not in the tree into a result (section 2.1).
* MUST document its presentation of results (section 2.2) and its limits (section 9).
* MUST reproduce Appendix A.



## Appendix A. Test Vectors

Evaluated against:

```
book
  chapter
    title First
    p one
    p two
  chapter
    title Second
    p three
list
  a
  b
  c
host
  '192.168.0.10' up
```

Results are written as the sequence of the selected nodes' names, with the text below each node in parentheses where it helps.

| Path | Result sequence |
|---|---|
| `book` | `book` |
| `book.chapter.title` | `title` (First) |
| `book.chapter{0}.title` | `title` (First) |
| `book.chapter{1}.title` | `title` (Second) |
| `book.chapter{9}.title` | empty |
| `book.chapter{}` | `chapter`, `chapter` |
| `book.chapter{}.title` | `title` (First), `title` (Second) |
| `book.chapter{}.title{}` | `title` (First), `title` (Second) |
| `book.chapter{}.p{}` | `p` (one), `p` (two), `p` (three) |
| `book.chapter.p{}` | `p` (one), `p` (two) |
| `book.chapter[0]` | `title` (First) |
| `book.chapter[1]` | `p` (one) |
| `list[0]` | `a` |
| `list[2]` | `c` |
| `list[9]` | empty |
| `[0]` | `book` |
| `host.'192.168.0.10'` | `192.168.0.10` (up) |
| `missing.x` | empty |

### A.1 Malformed Paths

Each MUST be rejected:

| Path | Reason |
|---|---|
| `` (empty) | not a path |
| `a.` | trailing separator |
| `a..b` | separator not between name elements |
| `a b` | space outside a quoted element |
| `a[]` | index with no number |
| `a[` | unterminated index |
| `a[x]` | index is not a number |
| `a{` | unterminated selector |
| `{0}` | selector with no preceding name element |
| `a[0]{1}` | selector preceded by an index |
| `a{}{0}` | selector preceded by a selector |
| `'abc` | unterminated quoted element |
| `a.'b\qc'` | undefined escape sequence |
| `a-b` | `-` is outside the token class; write `'a-b'` |
| `a+b` | `+` is outside the token class; write `'a+b'` |
| `a(b)` | held for the expression layer |
