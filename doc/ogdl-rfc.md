# RFC: Ordered Graph Data Language (OGDL)

## Status of This Memo

This document specifies the Ordered Graph Data Language (OGDL) as an informational specification in the style of an IETF RFC. It is intended to provide a precise, implementation-oriented definition of the language syntax and semantics.

This specification describes a character stream format that represents a strict ordered tree structure using indentation-sensitive layout.

Distribution of this memo is unlimited.



## Abstract

The Ordered Graph Data Language (OGDL) is a human-readable, layout-sensitive data language for representing strict ordered trees. Structure is expressed through indentation and line breaks rather than explicit delimiters. OGDL supports words, quoted strings, comments, and block continuations, and is defined over Unicode character streams. This document defines the lexical rules, grammar, indentation semantics, canonical serialization, and parsing model required for interoperable implementations.



## 1. Introduction

OGDL is a minimal, indentation-based data language designed for clarity, compactness, and deterministic tree representation. Unlike formats that rely on explicit structural delimiters, OGDL encodes hierarchy through indentation depth and line structure.

Key properties:

* Strict ordered tree model (no cycles, no shared nodes)
* Layout-sensitive syntax
* Unicode character model
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

Each line defines one or more *nested* nodes at a given indentation level.

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

is a pure ordered tree where each token is a node in the hierarchy. Any key/value interpretation is an application-level convention and is outside the scope of this specification.

OGDL itself is a structural tree language, not a mapping or key-value language.



## 3. Character Model

### 3.1 Unicode

OGDL operates on Unicode characters. Any encoding MAY be used provided it is decoded into Unicode characters before lexing.

OGDL uses Unicode solely as a universal character set abstraction. Except for control characters (Unicode General Category Cc)
and explicitly defined structural characters (SPACE, TAB, LF, CR), all Unicode characters are treated as opaque textual content
and have no intrinsic syntactic or semantic role.

Implementations MUST NOT treat any Unicode whitespace character other than U+0020 (SPACE) and U+0009 (TAB) as indentation, regardless of its visual rendering.

Authors SHOULD avoid the use of visually confusable whitespace or formatting characters (e.g., NO-BREAK SPACE, ZERO-WIDTH SPACE, bidirectional
control characters) in indentation-sensitive contexts, as they may lead to misleading visual layout while preserving different syntactic structure.

### 3.2 Character Classes

```
char         = any Unicode character
char_control = U+0000..U+001F | U+007F..U+009F
char_space   = U+0020 | U+0009 ; space or tab
char_break   = U+000A | U+000D ; LF or CR
char_end     = (char_control - char_space - char_break) | end of input
char_word    = char - char_break - char_end - char_space
char_string  = char - char_break - char_end
char_quoted  = char - char_end - "'" - '"'
```

Control characters other than TAB, LF, and CR terminate the stream.



## 4. Lexical Elements

### 4.1 Break

```
break = LF | CR | (CR LF)
```

### 4.2 Space and Indentation

```
space        = char_space+
space(n)     = char_space repeated n times
```

Indentation is layout-significant. Implementations MUST treat TAB and SPACE as distinct characters and MUST NOT mix them within the indentation prefix of a single line.

### 4.3 Word

```
word = (char_word - "'" - '"') char_word*
```

### 4.4 String

```
string = char_string+
```

### 4.5 Quoted Strings

```
quoted = ("'" quoted_char* "'") | ('"' quoted_char* '"')
quoted_char = char_quoted | escape
```

#### Escaping Rules

Within quoted strings only:

* `\\` represents a single backslash character
* `\'` represents a single apostrophe
* `\"` represents a single quotation mark
* Any other sequence starting with `\\` is interpreted literally (the backslash is preserved)

No other escape sequences are defined.

### 4.6 Comment

A valid comment satisfies ALL of the following:

* The `#` character appears in isolation
* It is preceded by a space or a break
* It is followed by at least one space

```
comment = (space | break) "#" space string (break | char_end)
```

Examples:

* `# comment` → valid
* `a#b` → not a comment
* `#abc` → not a comment



## 5. Grammar (EBNF)

```
element  = word | quoted
list     = element (space element)*
block(n) = space "\\" (comment | break) (space(m) string break)+ ; m > n
line(n)  = space(n) ((list block(n)?) | comment)
tree     = line* char_end
```



## 6. Indentation Semantics (Normative)

The EBNF grammar alone is insufficient to express OGDL structure. The following rules are normative and REQUIRED for interoperable implementations.

### 6.1 Leading Indentation Prefix

Only the leading sequence of indentation characters (U+0020 SPACE or U+0009 TAB) at the beginning of a line constitutes indentation. Any SPACE or TAB appearing after the first non-indentation character of a line is treated as content and has no structural meaning.

### 6.2 Indentation Depth

Indentation depth is defined as the number of leading indentation characters in a line.

Depth is computed lexically, NOT visually:

* A TAB counts as one indentation unit
* A SPACE counts as one indentation unit
* No tab-stop or column-width expansion is performed

### 6.3 Indentation Character Consistency

An OGDL stream MUST use exactly one indentation character for all indentation prefixes: either U+0020 (SPACE) or U+0009 (TAB).

The indentation character is implicitly determined by the first non-empty, non-comment line that contains indentation.

Once the indentation character has been determined:

* All subsequent indentation prefixes MUST use only that same character
* The presence of the other indentation character in any indentation prefix is a syntax error

### 6.4 Prohibition of Mixed Indentation

Mixing SPACE and TAB characters within the indentation prefix of a single line is always invalid.

Mixing SPACE and TAB indentation across different lines within the same stream is also invalid.

Examples of invalid indentation:

* A prefix containing both SPACE and TAB characters
* A SPACE-indented line followed by a TAB-indented line
* A TAB-indented line followed by a SPACE-indented line

### 6.5 Structural Rules

Parsers MUST interpret indentation as tree structure according to the following rules:

1. A child node MUST have strictly greater indentation depth than its parent line.
2. Sibling nodes MUST have identical indentation depth.
3. A decrease in indentation depth closes the current subtree until a matching depth is found.
4. Indentation levels MUST form a well-nested stack structure.

Parsers MUST maintain an indentation stack to construct the ordered tree.

### 6.6 Empty and Comment Lines

Empty lines and comment-only lines do not affect the indentation stack but MUST still respect the indentation character consistency rule if they contain indentation characters.

## 7. Canonical Serialization

The canonical textual representation of OGDL is defined as follows:

* Exactly two SPACE characters per indentation level
* One node per line
* No inline nesting
* No TAB characters for indentation
* Minimal quoting (only when required), using double quotes.
* Multi-line strings are double quoted (block is defined as non-canonical)

Example canonical form:

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

Non-canonical forms MAY use inline nesting and TAB indentation but remain valid if they respect the structural rules.



## 8. Block Continuations

A block begins with a backslash (`\\`) and allows multi-line string content indented more than the parent indentation level.

All block lines MUST:

* Have indentation strictly greater than the parent line
* End with a break



## 9. Error Handling

An implementation SHOULD treat the following as syntax errors:

* Mixed indentation characters within a stream or document
* Invalid indentation jumps (skipping parent levels inconsistently) (??)
* Unterminated quoted strings
* Control characters inside the stream that are not TAB, LF, or CR --> EOL?



## 10. Security Considerations

OGDL is a data description language and does not include executable constructs. However, implementations processing untrusted OGDL input should:

* Limit maximum indentation depth
* Limit maximum line length
* Validate Unicode input
* Protect against resource exhaustion in deeply nested trees



## 11. IANA Considerations

This document suggests the registration of the following media type:

```
text/ogdl
```

### 11.1 Media Type: text/ogdl

Type name: text
Subtype name: ogdl

Required parameters: none
Optional parameters: charset

OGDL is defined over Unicode character streams and is encoding-agnostic. Implementations MUST decode input into Unicode characters prior to parsing. The `charset` parameter MAY be used to indicate the character encoding used to transport the text, but is not semantically significant to the OGDL data model.

Encoding considerations: 8bit or binary. OGDL is textual but may contain any Unicode characters except restricted control characters as defined in this specification.

Security considerations: See Section 10.

Interoperability considerations: OGDL is indentation-sensitive and layout-dependent. Producers SHOULD emit canonical serialization using two SPACE characters per indentation level to ensure consistent interpretation across implementations.

Published specification: This document.

Applications that use this media type: Configuration systems, knowledge representation systems, logging formats, and hierarchical data interchange tools.

Fragment identifier considerations: Not defined by this specification.

Additional information:

* Magic number(s): none
* File extension(s): .ogdl
* Macintosh file type code(s): none

Person & email address to contact for further information: Not specified.

Intended usage: COMMON

Restrictions on usage: none

Author/Change controller: Not specified.

## 12. Conclusion

This specification defines OGDL as a Unicode, indentation-sensitive language for strict ordered tree representation,
with deterministic canonical serialization and minimal lexical complexity suitable for both human authoring and machine parsing.
