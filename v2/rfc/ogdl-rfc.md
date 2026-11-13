# RFC: Ordered Graph Data Language (OGDL)

## Status of This Memo

This document specifies the Ordered Graph Data Language (OGDL) as an informational specification in the style of an IETF RFC. It is intended to provide a precise, implementation-oriented definition of the language syntax and semantics.

This specification describes a character stream format that represents a strict ordered tree structure using indentation-sensitive layout.

This revision changes the meaning of lines that hold more than one element. Earlier drafts read `a b c` as a descending chain `a → b → c`; this revision reads it as a head `a` with the children `b` and `c`. An intermediate draft also gave `=` significance, as a pair `key=value`; that was withdrawn, and `=` is ordinary content. The earlier drafts were never registered or deployed, so this revision replaces them rather than coexisting with them (Section 10).

Distribution of this memo is unlimited.

The key words MUST, MUST NOT, REQUIRED, SHALL, SHALL NOT, SHOULD, SHOULD NOT, RECOMMENDED, MAY, and OPTIONAL in this document are to be interpreted as described in RFC 2119.



## Abstract

The Ordered Graph Data Language (OGDL) is a human-readable, layout-sensitive data language for representing strict ordered trees. Structure is expressed through indentation and line breaks rather than explicit delimiters. OGDL supports words, quoted strings, comments, and block continuations, and is defined over Unicode character streams. This document defines the lexical rules, grammar, tree construction algorithm, canonical serialization, conformance requirements, and error handling required for interoperable implementations.



## 1. Introduction

OGDL is a minimal, indentation-based data language designed for clarity, compactness, and deterministic tree representation. Unlike formats that rely on explicit structural delimiters, OGDL encodes hierarchy through indentation depth and line structure.

Key properties:

* Strict ordered tree model (no cycles, no shared nodes)
* Layout-sensitive syntax
* Unicode character model
* A small, closed set of syntactically significant characters
* Minimal escaping rules
* Canonical serialization defined

This specification does not define how bytes are converted into Unicode characters. Implementations MUST operate on a stream of Unicode characters.



## 2. Data Model

### 2.1 Tree Structure

An OGDL document represents a strict ordered tree where:

* Each node has zero or more ordered child nodes
* Nodes appear in document order
* Cycles are not allowed
* A node cannot have multiple parents

Each line defines one node, its **head**, at a given indentation level, and optionally children of that node written on the same line. The exact rule is normative and is given in Section 6.

### 2.2 Nodes, Names, and the Absence of Values

OGDL does NOT define a distinction between "names" and "values" as found in key-value data languages such as YAML.

Every element in OGDL is a node. A node consists of:

* A name (its textual content: word or quoted string)
* Zero or more child nodes

Child nodes are structural descendants and MUST NOT be interpreted as "values" of their parent node.

For example:

```
ip 192.168.0.10
```

represents a parent node `ip` with a child node `192.168.0.10`, not a key-value pair.

Similarly:

```
network
  eth0
    ip 192.168.0.10
```

is a pure ordered tree where each token is a node in the hierarchy. Any key/value interpretation is an application-level convention and is outside the scope of this specification. OGDL has no key/value syntax: `ip=192.168.0.10` is a single node with that name.

OGDL itself is a structural tree language, not a mapping or key-value language.

### 2.3 The Root Node

A document may contain several nodes at indentation depth zero. The data model therefore defines an implicit **root node** that is the parent of all depth-zero nodes.

* The root node is not written in the document and has no name. Implementations MAY give it any placeholder name; that name is not significant and MUST NOT be emitted by a canonical serializer.
* A document is well-formed if it contains zero or more depth-zero nodes. The empty document denotes a root node with no children.

### 2.4 Names

* Two sibling nodes MAY have the same name. OGDL is a tree, not a mapping; name uniqueness is never implied and MUST NOT be enforced by a parser.
* The empty string is a valid node name. It can only be written as an empty quoted string (`""` or `''`).
* A name is a sequence of Unicode characters. No normalization (case, Unicode NFC/NFD, whitespace trimming) is performed by a conforming parser. Two names are equal if and only if they are equal as Unicode character sequences.

### 2.5 Comments Are Not Data

Comments carry no structure and no content. They do not create nodes and do not appear in the data model, and canonical serialization emits none.

This does not oblige a parser to throw them away. A parser MAY retain comments, blank lines and layout *alongside* the tree, as presentation, so that a program can rewrite a document without losing what its author wrote. Retained presentation MUST NOT change the tree that is produced, and MUST NOT affect equivalence (Section 8.3). Appendix B gives an attachment rule that implementations retaining comments SHOULD follow, so that they agree on where a comment goes.



## 3. Character Model

### 3.1 Unicode

OGDL operates on Unicode characters. Any encoding MAY be used provided it is decoded into Unicode characters before lexing.

OGDL uses Unicode solely as a universal character set abstraction. Except for control characters (Unicode General Category Cc) and the syntactically significant characters listed in Section 3.3, all Unicode characters are treated as opaque textual content and have no intrinsic syntactic or semantic role.

Implementations MUST NOT treat any Unicode whitespace character other than U+0020 (SPACE) and U+0009 (TAB) as indentation, regardless of its visual rendering.

Authors SHOULD avoid the use of visually confusable whitespace or formatting characters (e.g., NO-BREAK SPACE, ZERO-WIDTH SPACE, bidirectional control characters) in indentation-sensitive contexts, as they may lead to misleading visual layout while preserving different syntactic structure.

Input that is not a valid sequence of Unicode scalar values — in particular unpaired surrogates (U+D800..U+DFFF) or malformed byte sequences in the transport encoding — MUST be rejected. Implementations MUST NOT silently substitute U+FFFD, because doing so can alter node names.

### 3.2 Character Classes

`EOS` denotes the end of the character stream. It is not a character; it is used in the productions below as a terminal marking stream exhaustion.

```
char         = any Unicode scalar value
char_control = U+0000..U+001F | U+007F..U+009F   ; Unicode General Category Cc
char_space   = U+0020 | U+0009                    ; space or tab
char_break   = U+000A | U+000D                    ; LF or CR
char_end     = char_control - char_space - char_break
char_word    = char - char_break - char_end - char_space
char_string  = char - char_break - char_end
char_quoted(q) = char - char_end - q - "\"   ; q is the delimiter in use
```

`char_quoted` is parameterized by the delimiter of the enclosing quoted string: only that delimiter and `\` are excluded. The other quote character is ordinary content (Section 4.4).

The ranges given for `char_control` are informative; the normative definition is Unicode General Category Cc.

A `char_end` character (any control character other than TAB, LF, and CR) MUST NOT appear in an OGDL stream. See Section 11.

### 3.3 Syntactically Significant Characters

The following characters, and only the following characters, have syntactic significance in OGDL:

| Character | Role |
|---|---|
| U+0020 SPACE | indentation, element separation |
| U+0009 TAB | indentation, element separation |
| U+000A LF, U+000D CR | line break |
| `"` U+0022, `'` U+0027 | quoted string delimiters |
| `\` U+005C | escape (inside quoted strings), block introducer |
| `#` U+0023 | comment introducer (in the positions defined in Section 4.6) |

Every other character — including `(`, `)`, `,`, `.`, `[`, `]`, `{`, `}`, `:`, `=` — is ordinary content and MAY appear unquoted anywhere within a word. No such character is reserved for future use by this specification.

This is a closed list. Notations layered on OGDL (path, query, or expression languages) operate on node *names* at the application level and MUST NOT be given lexical significance by a conforming parser.

### 3.4 Byte Order Mark

A U+FEFF appearing as the first character of the stream is a byte order mark and MUST be ignored.

A U+FEFF appearing anywhere else is ordinary content and is treated as part of a word or quoted string. Authors SHOULD avoid this.



## 4. Lexical Elements

### 4.1 Break

```
break = LF | CR LF | CR
```

The three forms are equivalent. A conforming parser MUST treat them identically; in particular, CR LF is a single break.

### 4.2 Space and Indentation

```
space  = char_space+
indent = char_space*
```

`indent` is the maximal run of `char_space` at the start of a line. `space` between elements separates them and has no structural meaning beyond that.

A **blank line** is a line consisting of zero or more `char_space` characters and nothing else, terminated by a `break` or by `EOS`. The number of spaces or tabs it contains is not significant anywhere in this specification: whitespace on a blank line indents nothing, so it is not an indentation prefix. Consequently a blank line neither determines the stream's indentation character nor can violate it (Sections 6.3, 6.4), and a blank line has no indentation depth. This keeps trailing whitespace left by editors from being an error.

Blank lines are ignored outside blocks (Section 6.6) and may be significant inside them (Section 7).

Indentation is layout-significant. Implementations MUST treat TAB and SPACE as distinct characters and MUST NOT mix them, either within one indentation prefix or across a stream (Section 6.3).

### 4.3 Word

```
word = (char_word - "'" - '"') char_word*
```

A word is a run of non-space, non-break characters that does not *begin* with a quote character. Quote characters are permitted inside a word and do not terminate it. Therefore:

* `don't` is one word.
* `a"b` is one word.
* `a"b c"` is two words, `a"b` and `c"` — not a word followed by a quoted string.
* `'abc` is not a word; it begins a quoted string (and is unterminated).

A backslash inside a word is an ordinary character; escape sequences exist only inside quoted strings. `C:\path\to` is one word denoting exactly those characters.

A word consisting of the single character `\` is the block introducer (Section 7) and is not a name. A word consisting of the single character `#` in a comment position (Section 4.6) introduces a comment and is not a name. To use either as a name, quote it.

### 4.4 Quoted Strings

```
quoted         = ('"' quoted_char('"')* '"') | ("'" quoted_char("'")* "'")
quoted_char(q) = char_quoted(q) | escape | break
```

A quoted string denotes exactly the characters it contains, after escape processing. The delimiters are not part of the name. The two delimiter characters are interchangeable; the choice carries no meaning. A quoted string delimited by `"` may contain `'` freely, and vice versa.

An unterminated quoted string is an error (Section 11).

The empty quoted string `""` denotes the empty name.

#### 4.4.1 Escaping

Escape sequences are recognized inside quoted strings only:

```
escape = "\" ( "\" | "'" | '"' | "n" | "t" )
```

| Sequence | Denotes |
|---|---|
| `\\` | U+005C REVERSE SOLIDUS |
| `\'` | U+0027 APOSTROPHE |
| `\"` | U+0022 QUOTATION MARK |
| `\n` | U+000A LINE FEED |
| `\t` | U+0009 CHARACTER TABULATION |

Any other sequence beginning with `\` inside a quoted string is an error. Undefined escapes are reserved so that the escape set can be extended by a future version of this specification without altering the meaning of existing documents.

#### 4.4.2 Line Breaks Inside Quoted Strings

A quoted string MAY span several lines. This form is accepted on input but is not canonical (Section 8): a canonical serializer writes `\n` instead.

When a `break` occurs inside a quoted string:

1. The break contributes a single U+000A LINE FEED to the name, whichever of LF, CR, or CR LF was written.
2. All `char_space` characters immediately following that break are removed and contribute nothing.

Rule 2 exists so that a multi-line string can be indented to match its surroundings. Its consequence is that a line *inside* a multi-line quoted string cannot begin with a space or tab; `\t`, or a preceding `\n` escape, MUST be used to express that.

Example — the two forms below denote the same name:

```
text "line one
      line two"

text "line one\nline two"
```

### 4.5 String

```
string = char_string+
```

A `string` is an unstructured run of characters terminated by a break. It is used only as the content of a block line (Section 7) and as the text of a comment. It is never a node name by itself.

### 4.6 Comment

A `#` character introduces a comment if and only if ALL of the following hold:

* It is the first character of a line, or is immediately preceded by a `char_space`.
* It is immediately followed by a `char_space`, a `break`, or `EOS`.

```
comment = "#" [ space [ string ] ]
```

A comment extends to the end of the line. The break that terminates it is not part of the comment and still terminates the line.

Examples:

| Input | Result |
|---|---|
| `# comment` | comment |
| `a b # comment` | word `a`, word `b`, comment |
| `#` at end of line | comment (empty) |
| `a#b` | one word `a#b` — `#` not preceded by a space |
| `#abc` | one word `#abc` — `#` not followed by a space |
| `a #b` | words `a` and `#b` |

A comment MAY appear on a line that also contains elements, in which case it follows them. A comment-only line does not affect tree structure (Section 6.6).



## 5. Grammar (EBNF)

The following grammar defines the token-level shape of a document. It is deliberately context-free and therefore CANNOT express the indentation constraints that determine tree structure, nor the positional constraints on `#`. Sections 6 and 7 are normative for those; where this grammar and Section 6 appear to differ, Section 6 governs.

```
document   = [ U+FEFF ] line* EOS

line       = indent [ content ] [ comment ] line_end
line_end   = break | EOS
content    = ( list [ space block ] ) | block
list       = element ( space element )*
element    = word | quoted

block      = "\" [ space comment ] break block_line*
block_line = indent [ string ] line_end
```

Notes:

* `content` is optional, so a blank line (Section 4.2) is valid at any point.
* A `block` MAY be the entire content of a line. Section 7 defines where its content attaches in both cases.
* Whether a `#` begins a `comment` or is part of a `word` is decided by Section 4.6, not by this grammar.
* Adjacent elements need not be separated by `space` when the first ends with a quote delimiter: `"a"b` is the quoted string `a` followed by the word `b`.



## 6. Tree Construction (Normative)

### 6.1 Leading Indentation Prefix

Only the leading sequence of indentation characters (U+0020 SPACE or U+0009 TAB) at the beginning of a line constitutes indentation. Any SPACE or TAB appearing after the first non-indentation character of a line is element separation or content, and has no structural meaning.

### 6.2 Indentation Depth

Indentation depth is the number of leading indentation characters in a line.

Depth is computed lexically, NOT visually:

* A TAB counts as one indentation unit
* A SPACE counts as one indentation unit
* No tab-stop or column-width expansion is performed

### 6.3 Indentation Character Consistency

An OGDL stream MUST use exactly one indentation character throughout: either U+0020 (SPACE) or U+0009 (TAB).

The indentation character is determined by the first line in the stream that has a non-empty indentation prefix. Comment-only lines and block lines count, since their indentation indents something. Blank lines (Section 4.2) do not: their whitespace is not an indentation prefix, so a blank line neither determines the indentation character nor is constrained by it.

Once determined:

* Every subsequent indentation prefix MUST use only that character.
* The presence of the other indentation character in any indentation prefix is an error.

### 6.4 Prohibition of Mixed Indentation

Mixing SPACE and TAB characters within the indentation prefix of a single line is always invalid, even in the line that determines the indentation character.

Mixing SPACE and TAB indentation across different lines of the same stream is also invalid.

Examples of invalid indentation:

* A prefix containing both SPACE and TAB characters
* A SPACE-indented line followed by a TAB-indented line
* A TAB-indented line followed by a SPACE-indented line

Blank lines are exempt from both rules, per Section 4.2. A blank line containing a TAB in an otherwise SPACE-indented stream is not an error, because it indents nothing.

### 6.5 Line Structure

The first element of a line's `list` is the line's **head**. The remaining elements are the head's **arguments**: they become children of the head, in order, as siblings of one another.

```
rule allow 10.0.0.0/8
```

is `rule` with the two children `allow` and `10.0.0.0/8`. Every argument is a leaf.

The head is the line's only **open** node. A following line with greater indentation attaches its head to it, after the arguments:

```
a b c
  d
```

yields `a` with the children `b`, `c` and `d`. Nesting deeper than one level therefore always takes a line per level. This is the reading of the command line — a name followed by its arguments. The same rule applies at every depth: a line holds one node and its leaf children, and every further level of structure is a further line. A record is therefore written one field per line:

```
S1
  value 1.0
  class asil
```

which denotes the same tree as

```
S1
  value
    1.0
  class
    asil
```

whereas `S1 value 1.0 class asil` is `S1` with four leaf children. No marker within a line introduces a deeper level.

### 6.6 Construction Algorithm

A conforming parser MUST build the tree as follows.

Maintain a stack of (node, depth) pairs, initialized with the root node (Section 2.3) at depth `-1`, lower than any indentation depth. Only heads are ever pushed.

For each line, in document order, excluding lines consumed by a block (Section 7):

1. Skip the line entirely if it has no `content` — that is, if it is a blank line or comment-only. Such lines never modify the stack, and a blank line does not count as the "preceding content-bearing line" in Section 6.7. (Comment-only lines remain subject to Sections 6.3 and 6.4; blank lines do not, per Section 4.2.)
2. Apply the dedent check of Section 6.7.
3. Let `n` be the line's indentation depth. Pop the stack while the depth at its top is greater than or equal to `n`. Let `P` be the node now at the top.
4. If the line has a `list`:
   a. Append a new node for the head as the last child of `P`.
   b. Push the head with depth `n`.
   c. For each argument, in order, append a new leaf node as the last child of the head. Arguments are not pushed.
5. If the line ends with a block (Section 7), the block's content becomes one further leaf node: the last child of the head if the line has a `list`, otherwise the last child of `P`. It is not pushed, so nothing can attach beneath it.

The stack after the last line is discarded; the tree is reachable from the root.

### 6.7 Dedent Must Land on an Open Level

Let `D` be the set of indentation depths of the heads currently on the stack, together with depth 0, which is always open because the root is the parent of every depth-zero head.

When a line's indentation depth `n` is **less** than that of the preceding content-bearing line, `n` MUST be a member of `D`. Otherwise the stream is ill-formed.

This rejects a dedent onto a level that was never opened:

```
a
    b
  c        <- error: depth 2 was never opened; open depths are {0, 4}
```

Implementations MUST NOT silently reinterpret such a line, for instance by treating `c` as a sibling of `b`.

### 6.8 Structural Consequences

The following follow from Sections 6.5 to 6.7 and are stated for clarity:

1. A line contributes at most two levels of structure below its parent: the head, and the head's arguments, which are leaves.
2. Sibling heads written on separate lines have identical indentation depth.
3. A decrease in indentation depth closes subtrees until a matching open level is found.
4. Indentation levels form a well-nested stack.
5. Indentation depth may increase by any amount from one line to the next; only *decreases* are constrained (Section 6.7).



## 7. Block Continuations

A block carries multi-line text without quoting.

A block is introduced by a word consisting of exactly one `\` character appearing as the last element of a line. It MAY be followed on that line only by an optional comment, then a break.

The **base indentation** of a block is the indentation depth of its first non-blank line. That line MUST have indentation depth strictly greater than the line introducing the block.

The block extends to, but does not include, the first non-blank line whose indentation depth is less than the base indentation, or to `EOS`. Blank lines never terminate a block.

A block therefore consists of:

* every non-blank line with indentation depth greater than or equal to the base indentation, and
* every blank line that lies between two such lines.

Blank lines preceding the first non-blank line of a block, and blank lines following its last non-blank line, are not part of the block. A block that contains no non-blank line has no content and denotes the empty name.

Block lines are consumed by the block. They are not processed as lines by Section 6.6 and never affect the stack; their content is opaque and is not lexed into elements, comments, or blocks. Non-blank block lines remain subject to Sections 6.3 and 6.4, since their indentation prefixes are indentation; blank lines are exempt (Section 4.2).

The block's content is produced by taking each of its lines in order, and joining the results with a single U+000A:

* A non-blank line contributes its characters with exactly `base` indentation characters removed from the start. Indentation beyond `base` is preserved as content, so relative indentation inside a block is significant and retained.
* A blank line contributes the empty string, whatever whitespace it contains. Its own width is discarded rather than measured against `base`, so a blank line inside a block is never an error and never contributes trailing whitespace.

A trailing break at the end of the last block line is not part of the content.

The content becomes a single node, a leaf. If the introducing line holds elements, the node is appended as the last child of the line's head, after its arguments. If the line's content is a bare `\`, the node is appended at the line's own indentation depth, as a sibling of the nodes written by neighbouring lines at that depth. Section 6.6, step 5, states this uniformly.

Example:

```
description \
  some long text
    indented further
  and some more.
```

yields `description → "some long text\n  indented further\nand some more."`.

Example with an interior blank line — the blank line separates two indented paragraphs and is retained, so the block does not end at it:

```
text \
  first paragraph

  second paragraph
next
```

yields `text → "first paragraph\n\nsecond paragraph"` and a sibling node `next`. The blank line would be retained equally if it held twenty spaces or none.

Blocks are not canonical (Section 8): a canonical serializer writes the same node as a quoted string with `\n` escapes.



## 8. Canonical Serialization

Canonical serialization is a total function from a tree to a character stream. It is the basis of the equivalence rule in Section 8.3.

### 8.1 Rules

A canonical OGDL stream:

* Contains no byte order mark.
* Uses U+000A LF as its only break.
* Uses exactly two U+0020 SPACE characters per indentation level, and no TAB.
* Emits exactly one node per line: no arguments. A node `a` with a child `b` is written on two lines, not as `a b`.
* Ends every line, including the last, with a single LF. An empty tree serializes to the empty stream.
* Contains no trailing space or tab on any line, no empty lines, no comments, and no blocks.
* Does not emit the root node itself; its children are the depth-zero lines.
* Quotes minimally (Section 8.2), and always with `"`.

### 8.2 Minimal Quoting

A name is written unquoted if and only if it is a valid `word` (Section 4.3) and is not one of the reserved single-character forms. Concretely, a name MUST be quoted if any of the following holds:

* it is empty;
* it contains SPACE, TAB, LF, or CR;
* its first character is `"` or `'`;
* it is exactly `\`;
* it is exactly `#`.

Otherwise it MUST NOT be quoted. In particular a name containing `\` in a non-initial position, such as `C:\path`, is written unquoted; `#` in a non-initial position is written unquoted; and `a=b`, `a==b` and `S>=S2`, which are words, are written unquoted.

A name containing a `char_end` character (Section 3.2) has no OGDL representation, quoted or otherwise, because such characters MUST NOT appear in a stream. A producer given such a name MUST report an error rather than emit it, drop it, or substitute for it.

When quoted, the following characters, and only these, are escaped: `\` as `\\`, `"` as `\"`, LF as `\n`, TAB as `\t`. `'` is never escaped, since the delimiter is always `"`. A canonical line therefore never contains a raw LF or TAB inside a name, which preserves the one-node-per-line invariant.

### 8.3 Equivalence

Two OGDL streams denote the same tree if and only if their canonical serializations are identical character sequences.

A canonical stream reparses to the tree it was produced from; canonicalization is idempotent.

Non-canonical forms — arguments, TAB indentation, blocks, multi-line quoted strings, single quotes, comments, blank lines — remain valid input and MUST be accepted, but carry no additional meaning.

### 8.4 Example

Input:

```
network
  eth0
    ip 192.168.0.10
    mask 255.255.255.0
    gw   192.168.0.1
hostname crispin
```

Canonical form:

```
network
  eth0
    ip
      192.168.0.10
    mask
      255.255.255.0
    gw
      192.168.0.1
hostname
  crispin
```

### 8.5 Compact Form

Canonical form is the basis of equality, but it spends a line on every node, which is a poor default for files that people read. A producer MAY instead emit the **compact form**, which is deterministic and reparses to the same tree, but is not canonical and MUST NOT be used for equivalence.

The compact form follows Section 8.1, except for one-node-per-line. Each node is written as a head, followed on the same line by its **inline run**, and its remaining children are written on the following lines, one level deeper. The inline run is computed as follows:

1. Take the longest prefix of the node's children that are leaves, each written as a name, quoted per Section 8.2.
2. If that prefix has at most one child, it is the inline run.
3. Otherwise it is the inline run if the resulting line, indentation included, is at most 80 characters long; if not, the inline run is empty.

For the example of Section 8.4 this gives

```
network
  eth0
    ip 192.168.0.10
    mask 255.255.255.0
    gw 192.168.0.1
hostname crispin
```



## 9. Conformance

### 9.1 Documents and Streams

A **stream** is a single OGDL **document**. This specification defines no document separator and no concatenation semantics; an application requiring several documents MUST provide its own framing.

A document is **well-formed** if it satisfies Sections 3 through 7. This specification defines no notion of validity beyond well-formedness. Validity against a schema is defined separately, by the OGDL Schema specification (`ogdl-schema-rfc.md`), which operates on the tree and changes nothing defined here.

### 9.2 Conforming Parser

A conforming parser:

* MUST accept every well-formed document and produce the tree defined by Section 6.
* MUST reject every ill-formed document, reporting an error as described in Section 11. Rejection MUST be the default behaviour.
* MUST NOT normalize, reorder, deduplicate, or reinterpret node names.
* MAY additionally offer a lenient mode that recovers from the errors of Section 11. Lenient mode MUST NOT be the default, and its recoveries are outside the scope of this specification.
* MUST document any implementation limits it imposes (Section 12).

### 9.3 Conforming Producer

A conforming producer emits canonical serialization (Section 8) unless directed otherwise by its user.

A producer that emits non-canonical output MUST still emit well-formed output.

### 9.4 Test Vectors

Appendix A gives input/canonical-output pairs. A conforming implementation MUST reproduce them.



## 10. Versioning and Extensibility

This specification defines no in-band version marker, and none is to be inferred from document content.

The following extension points are open, and a future version MAY use them without invalidating documents that conform to this one:

* Additional escape sequences inside quoted strings. Undefined escapes are errors today (Section 4.4.1), which reserves the space.
* Fragment identifier syntax for the `text/ogdl` media type (Section 13).

Conversely, the set of syntactically significant characters (Section 3.3) is closed and *no* character is held in reserve. Any future change that gives lexical significance to a character that is content today — for example an inline grouping syntax using `(`, `)` and `,` — would change the meaning of existing documents. Such a change is therefore out of scope for this specification and MUST NOT be introduced in place: it would require a distinct media type or an explicit media type parameter, so that a recipient can tell the two languages apart.

This revision is itself such a change with respect to earlier drafts: it reads a multi-element line as a head with arguments rather than as a chain (Section 6.5), and it withdraws the significance an intermediate draft gave to `=`. It is introduced in place only because the earlier drafts were never registered and have no deployed base. Once `text/ogdl` is registered, the rule above applies without exception.



## 11. Error Handling

A conforming parser MUST treat the following as errors and MUST NOT produce a tree from a document containing them:

* An indentation prefix containing both SPACE and TAB characters (Section 6.4).
* An indentation prefix using a character other than the one determined for the stream (Section 6.3).
* A dedent to an indentation depth that was never opened (Section 6.7).
* An unterminated quoted string, including one terminated by `EOS`.
* An undefined escape sequence inside a quoted string (Section 4.4.1).
* A `char_end` character anywhere in the stream (Section 3.2).
* Input that is not a valid sequence of Unicode scalar values (Section 3.1).
* A block whose base indentation is not strictly greater than that of its introducing line (Section 7).
* Exceeding a documented implementation limit (Section 12).

An error report SHOULD identify the line and, where meaningful, the character offset within the line.

Earlier drafts of this specification left the dedent case undecided and stated these conditions as SHOULD. Both are now normative as written above.



## 12. Security Considerations

OGDL is a data description language and does not include executable constructs. It has no include directive, no reference or anchor mechanism, and no type coercion, so the classes of attack associated with such features do not apply.

Implementations processing untrusted OGDL input SHOULD nevertheless:

* Impose and document a maximum nesting depth. Nesting is bounded only by indentation, so a small input can request very deep recursion; a recursive-descent parser MUST guard against stack exhaustion.
* Impose and document a maximum line length and maximum name length. A single quoted string or block can otherwise consume unbounded memory.
* Impose and document a maximum total input size.
* Reject invalid Unicode rather than substituting replacement characters (Section 3.1), since substitution can cause two distinct inputs to yield equal names.
* Ensure that lexing and tree construction are linear in input size; neither is inherently superlinear, and an implementation that is should be treated as defective.

Applications SHOULD be aware that node names may contain bidirectional control characters and visually confusable characters, so a name's rendered appearance is not a reliable guide to its content (Section 3.1).



## 13. IANA Considerations

This document suggests the registration of the following media type:

```
text/ogdl
```

### 13.1 Media Type: text/ogdl

Type name: text

Subtype name: ogdl

Required parameters: none

Optional parameters: `charset`. The default value is `UTF-8`. For interchange, UTF-8 MUST be used; the parameter exists to describe transport of streams in another encoding, and is not semantically significant to the OGDL data model, which is defined over Unicode characters.

Encoding considerations: 8bit. OGDL is textual and line-oriented, and may contain any Unicode characters except the control characters restricted in Section 3.2.

Security considerations: See Section 12.

Interoperability considerations: OGDL is indentation-sensitive and layout-dependent. Producers SHOULD emit canonical serialization (Section 8) — two SPACE characters per indentation level, LF breaks, one node per line — or the compact form (Section 8.5) to ensure consistent interpretation across implementations.

Published specification: This document.

Applications that use this media type: Configuration systems, knowledge representation systems, logging formats, and hierarchical data interchange tools.

Fragment identifier considerations: No fragment identifier syntax is defined by this specification. The syntax is reserved for a future specification defining node addressing.

Additional information:

* Magic number(s): none
* File extension(s): .ogdl
* Macintosh file type code(s): TEXT

Person & email address to contact for further information: TBD before publication.

Intended usage: COMMON

Restrictions on usage: none

Author/Change controller: TBD before publication.



## 14. Conclusion

This specification defines OGDL as a Unicode, indentation-sensitive language for strict ordered tree representation, with a closed set of significant characters, a deterministic tree construction algorithm, and a canonical serialization that provides a decidable equivalence relation, suitable for both human authoring and machine parsing.



## Appendix A. Test Vectors

Each case gives an input document and its canonical serialization. `·` is not part of the data; it is used only where leading spaces would otherwise be invisible. Canonical forms end with a final LF.

### A.1 Head and arguments

Input:

```
a b c
```

Canonical:

```
a
  b
  c
```

### A.2 Indented line attaches to the head

Input:

```
a b c
  d
```

Canonical:

```
a
  b
  c
  d
```

### A.3 Siblings and dedent

Input:

```
a
  b x
    c f
  d
e
```

Canonical:

```
a
  b
    x
    c
      f
  d
e
```

### A.4 Quoting

Input:

```
a 'hola hola'
b "with \"quotes\""
c ""
d C:\path\to
```

Canonical:

```
a
  "hola hola"
b
  "with \"quotes\""
c
  ""
d
  C:\path\to
```

### A.5 Multi-line quoted string

Input:

```
text "line one
      line two"
```

Canonical:

```
text
  "line one\nline two"
```

### A.6 Block

Input:

```
description \
  some long text
    indented further
  and some more.
```

Canonical:

```
description
  "some long text\n  indented further\nand some more."
```

### A.6b Block with interior blank lines

Input — `·` marks spaces on otherwise-blank lines, and is not part of the data:

```
text \
  first paragraph
··········
  second paragraph

next
```

Canonical:

```
text
  "first paragraph\n\nsecond paragraph"
next
```

The ten-space blank line is interior to the block, so it is retained as an empty line and its width is discarded. The final blank line trails the block's last non-blank line, so it is dropped, and `next` ends the block.

### A.7 Comments and blank lines

Input:

```
# leading comment

a b   # trailing comment
#
a#b
```

Canonical:

```
a
  b
a#b
```

### A.8 Duplicate names

Input:

```
a 1
a 2
```

Canonical:

```
a
  1
a
  2
```

### A.9 Empty document

Input: the empty stream, or a stream of blank and comment-only lines.

Canonical: the empty stream.

### A.10 `=` is content

Input:

```
w a=b S>=S2 --flag=x a= =b a==b x.y=1 url=http://x?a=b k=\ "k=v"
```

Canonical:

```
w
  a=b
  S>=S2
  --flag=x
  a=
  =b
  a==b
  x.y=1
  url=http://x?a=b
  k=\
  k=v
```

`=` has no significance anywhere: each of these is one word, and none needs quoting. `"k=v"` is written unquoted in canonical form.

### A.11 A record takes a line per field

Input:

```
S1 value 1.0 class asil
S2
  value 2.0
  class asil
```

Canonical:

```
S1
  value
  1.0
  class
  asil
S2
  value
    2.0
  class
    asil
```

A line never contributes more than one level below its head, so `S1`'s four arguments are siblings.

### A.12 Block after arguments

Input:

```
text en \
  hello
```

Canonical:

```
text
  en
  hello
```

### A.13 Ill-formed inputs

Each of the following MUST be rejected:

| Input | Reason |
|---|---|
| `a` / `····b` / `··c` | dedent to an unopened level (Section 6.7) |
| `a` then a TAB-indented `b`, in a SPACE-indented stream | indentation character inconsistency (Section 6.3) |
| A line indented with one SPACE and one TAB | mixed indentation prefix (Section 6.4) |
| `a "unterminated` | unterminated quoted string |
| `a "bad \q escape"` | undefined escape sequence |
| `a \ x` | block introducer not last on its line (Section 7) |



## Appendix B. Comment Attachment (Informative)

A parser that retains comments (Section 2.5) needs to decide which node each comment belongs to, so that a program can move, delete or keep it with that node. The following rule is simple, preserves order, and reproduces a document whose layout is already normalized byte for byte.

Each comment, and each blank line, is attached to a node in one of three places: **before** the node, on the node's **line** (a trailing comment), or **after** the node's last child, indented as its children.

1. A trailing comment belongs to the head of its line, or to the block of a bare block line.
2. Comment-only and blank lines are queued until the next content-bearing line, at depth `n`, or the end of the stream, where `n` is taken as 0.
3. Then, in order, each queued comment indented deeper than `n` goes *after* the deepest open head that the new line closes (whose depth is at least `n`) and that is shallower than the comment. A comment never goes to a deeper head than the one before it, so order is kept.
4. The first queued comment that has no such head, and everything queued after it, goes *before* the new line's head; at the end of the stream, after the last node at depth zero.
5. A blank line goes with the comment that follows it, or before the next head if no comment follows. Blank lines at the start and end of the stream are dropped, and consecutive blank lines count as one.

Example — each comment's owner is given on the right:

```
a
  b
    c
    # end of b                 after b
  # end of a                   after a
# before d                     before d
d   # about d                  line of d
```

A comment is never a child, so retaining comments cannot change the tree, and a comment attached to a node that a program removes is removed with it.
