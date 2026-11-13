# RFC: OGDL Expressions

## Status of This Memo

This document specifies the OGDL expression language and its evaluation, in the style of an IETF RFC.

It realises the extension point held open by `ogdl-path-rfc.md` section 10 (the "path RFC"), and depends on that document and on `ogdl-rfc.md` (the "core specification").

No prior specification of this layer exists. The reference implementation, `github.com/rveen/ogdl`, has evaluated expressions for years to drive its template engine; this document specifies that capability, and departs from it where its behaviour was accidental rather than designed. Section 12 lists every departure.

Distribution of this memo is unlimited.

The key words MUST, MUST NOT, REQUIRED, SHALL, SHALL NOT, SHOULD, SHOULD NOT, RECOMMENDED, MAY, and OPTIONAL in this document are to be interpreted as described in RFC 2119.



## Abstract

An OGDL expression computes a value in the context of an OGDL tree. Expressions combine literals, paths, arithmetic, comparison and boolean operators, and calls to functions and methods provided by the host program. This document defines the lexis, grammar, value model, coercion and evaluation rules, the interface to host values, and the error and security model.



## 1. Introduction

The core specification defines a tree of text. The path RFC defines how to address nodes in it. Expressions are the third layer: they compute *with* what a path addresses.

```
2 + 3 * 4
user.age >= 18
config.timeout
user.FullName()
!(a && b)
```

An expression is a standalone character string, like a path and unlike a stream. Evaluation takes an expression and a **context node**, and produces a value.

### 1.1 Relationship to the Path RFC

The path RFC keeps the unquoted token class narrow — letters, digits and underscore — expressly so that operator characters stay free for this document (path RFC sections 4.2 and 10). That reservation is now spent, and no further extension of that kind is possible: the operator characters defined in section 3.3 below are now committed.

This document also extends paths themselves, in ways the path RFC anticipated:

* An index may contain an expression, not only a number (path RFC section 4.4).
* A path element may be a **call**, `name(args)`.
* A path element may be a **group**, `.(expr)`, whose value is used as the element's name.

These extensions apply only where an expression is being parsed. An implementation MUST continue to accept the strict path language of the path RFC as its own language, unchanged: `Get` and its equivalents parse strict paths, in which `(` and `)` remain errors.



## 2. Value Model

### 2.1 Values

An expression evaluates to one of:

| Kind | Notes |
|---|---|
| integer | 64-bit signed |
| float | 64-bit IEEE 754 |
| string | a sequence of Unicode characters |
| boolean | |
| node | a `*Node`, the result of a path that addresses part of the tree |
| host value | any value the host program placed in the tree (section 8) |
| nil | the absence of a value |

A stream carries only text (core specification, section 2.4). Integers, floats and booleans therefore never arise from parsing a document; they arise from literals in an expression, from operators, from the host, or from the coercion rules below.

### 2.2 Operand Simplification

Where a value is used as an **operand** — of an operator, as a call argument, or as the result of an expression — a node is first simplified:

> A node with no children simplifies to its own content. A node with exactly one child, that child having no children of its own, simplifies to that child's content. Any other node is used as it stands.

This is what makes the common case read naturally. Given

```
a
  b 1
```

`a.b` addresses the node `b`, whose single leaf child holds `1`, so `a.b + 1` is `2`.

Simplification applies once, not repeatedly, and never to a node with several children or with a grandchild.

### 2.3 Coercion

Coercion happens at the operator, never in advance. The rules are total: they either produce a value of the required kind, or report that they cannot.

**To a number.** An integer or float is itself. A string is a number if it parses completely as a decimal integer or float (section 3.1), optionally preceded by `+` or `-` — the forms of the schema RFC, section 4.1; otherwise it is not a number. Hexadecimal forms, digit separators, `Inf` and `NaN` are not numbers. A boolean, node or host value is not a number. A node simplifies first (section 2.2).

**To a string.** Every value has a textual form: the node name for a node (core specification, section 8.2), the decimal form for numbers, `true` or `false` for booleans, the empty string for nil, and the host's own rendering for a host value.

**To a boolean.**

| Value | Result |
|---|---|
| boolean | itself |
| integer, float | false if zero, otherwise true |
| `"true"`, `"false"` | the corresponding boolean |
| any other string | false if empty, otherwise true |
| nil | false |
| node | false if it holds no content and has no children, otherwise true |
| host value | false if it is a nil pointer or interface, otherwise true |

Coercion to a boolean is total and never fails; there is no error case.



## 3. Lexis

### 3.1 Numbers

```
number   = integer | float
integer  = DIGIT+
float    = DIGIT+ "." DIGIT* exponent? | DIGIT+ exponent
exponent = ("e" | "E") ("+" | "-")? DIGIT+
```

`DIGIT` is U+0030..U+0039. Other Unicode digits are not numbers, following the path RFC section 4.1.

A number literal carries no sign: `-1` is the unary minus operator applied to `1` (section 4.2). An integer literal that does not fit in 64 bits is an error.

### 3.2 Strings

A string literal is a quoted element with exactly the syntax and escapes of the path RFC section 4.3: `'…'` or `"…"`, with `\\`, `\'` and `\"`.

The two delimiters are interchangeable and carry no meaning.

### 3.3 Operators

```
op_unary  = "!" | "-"
op_binary = "*" | "/" | "%" | "+" | "-"
          | "==" | "!=" | ">=" | "<=" | ">" | "<"
          | "&&" | "||"
          | op_assign
op_assign = "=" | "+=" | "-=" | "*=" | "/=" | "%="
```

Operators are lexed by longest match, so `!=` is one operator and never `!` followed by `=`, and `+=` is one operator and never `+` followed by `=`.

A single `&` or `|` is an error, reserved for a possible future bitwise layer (section 11).

### 3.4 Boolean Literals

The words `true` and `false` are boolean literals.

They are also well-formed path tokens, and this is the one place where the two lexes collide. A bare `true` or `false` in an expression is the boolean literal. To address a node named `true`, quote it: `'true'`. Implementations MUST NOT resolve a bare `true` or `false` as a path.

### 3.5 Whitespace

U+0020 SPACE and U+0009 TAB may appear between tokens and are not significant. A line break MUST NOT appear in an expression.



## 4. Grammar

```
expression = binary
binary     = unary ( op_binary unary )*
unary      = op_unary unary | primary
primary    = literal | path | "(" expression ")"
literal    = number | string | "true" | "false"
path       = extended path (section 5)
```

### 4.1 Precedence and Associativity

`binary` is resolved by precedence.

| Precedence | Operators | Associativity |
|---|---|---|
| 5 (tightest) | `*` `/` `%` | left |
| 4 | `+` `-` | left |
| 3 | `==` `!=` `>=` `<=` `>` `<` | left |
| 2 | `&&` | left |
| 1 | `\|\|` | left |
| 0 (loosest) | `=` `+=` `-=` `*=` `/=` `%=` | **right** |

This is Go's precedence with the bitwise operators removed, and with assignment — a statement in Go — admitted as an operator at the loosest level.

Assignment is right-associative, so `a = b = 1` assigns `1` to `b` and then to `a`. Every other operator is left-associative, so `1-2+3` is `(1-2)+3`.

Because assignment binds loosest, `a = 1 + 2` assigns `3`, and `a = b == c` assigns a boolean.

### 4.2 Unary Operators

`!` coerces its operand to a boolean (section 2.3) and negates it.

`-` negates a number; applied to a value that is not a number, it is an error. `-` binds tighter than any binary operator, so `-a.b * 2` is `(-a.b) * 2`.

Unary operators may be stacked: `!!x`, `--x`.



## 5. Extended Paths

Within an expression, a path is the language of the path RFC section 5 with three additions.

```
tail = ( "." name_element ) | index | selector | call | group
index = "[" expression "]"
call  = "(" arg_list? ")"
group = "." "(" expression ")"
arg_list = expression ( "," expression )*
```

### 5.1 Expression Index

`[expr]` evaluates `expr`, coerces it to an integer, and indexes as the path RFC section 6.3 defines. A negative index, or one that is not a number, is an error. `a[0]` therefore keeps exactly its meaning; only what may appear between the brackets is widened.

### 5.2 Call

`name(args)` calls the value the preceding path selected, with the given arguments, and continues the path with the result (section 8). The argument list may be empty. A trailing comma is an error.

A call MUST be preceded by a path element. A leading `(` is a parenthesised expression (section 4), not a call.

### 5.3 Group

`.(expr)` evaluates `expr`, takes its textual form, and uses that as the name of a path element. It is the indirection operator: given

```
a
  c 1
b c
```

`a.(b)` evaluates `b` to `c` and then selects `a.c`, yielding `1`.

An expression that evaluates to the empty string selects nothing.

The dot before `(` is REQUIRED, and is what distinguishes a group from a call: `a(b)` calls `a`, `a.(b)` indexes into `a` by the value of `b`.



## 6. Evaluation

Evaluation takes an expression and a context node, and yields a value or an error.

### 6.1 Paths

A path is evaluated against the context node as defined by the path RFC section 6, extended by section 5 above and by section 8 for host values.

A path that matches nothing evaluates to nil. This is not an error: it is the same distinction the path RFC section 8 draws between a malformed path and an empty result.

### 6.2 Arithmetic: `+` `-` `*` `/` `%`

Both operands are simplified (section 2.2) and coerced to numbers (section 2.3).

* If both are numbers, and both are integers, the result is an integer.
* If both are numbers and either is a float, both are converted to float and the result is a float.
* Otherwise, for `+` only, both operands are converted to strings and the result is their concatenation.
* Otherwise it is an error.

`/` and `%` with a zero divisor are an error. `%` requires two integers; applied to a float it is an error.

A nil operand — typically a path that matched nothing — is an error for every arithmetic operator, `+` included. Concatenating it as an empty string would turn a mistyped path into a plausible result instead of a diagnosable one.

Integer overflow wraps, as in Go.

### 6.3 Comparison: `==` `!=` `>=` `<=` `>` `<`

Both operands are simplified and coerced to numbers.

* If both are numbers, they are compared numerically, converting to float if either is a float.
* Otherwise, if both are booleans, `==` and `!=` compare them and the ordering operators are an error.
* Otherwise both are converted to strings and compared lexicographically by Unicode code point.

The result is always a boolean.

Comparison is symmetric: both operands are examined before a kind is chosen. An implementation MUST NOT let the first operand alone determine the comparison kind.

### 6.4 Boolean: `&&` `||`

Operands are coerced to booleans (section 2.3), which cannot fail.

Both operators **short-circuit**: `&&` evaluates its right operand only if the left is true, `||` only if the left is false. A right operand that is not evaluated cannot produce an error, and any call it contains is not made.

### 6.5 Assignment: `=` `+=` `-=` `*=` `/=` `%=`

Assignment writes into the tree. It is the only construct in this document that does, calls aside (section 7).

**The target.** The left operand MUST be a path (section 5). Assigning to a literal, to an operator expression, or to the result of a call is an error. The path MUST address exactly one node; an empty selector `{}`, which addresses a set, is not a target.

Parentheses only group, so `(a) = 1` assigns to `a`, as it does in Go.

**Creation.** A target path may name nodes that do not yet exist. Every missing name element is created, in order, so that `a.b.c = 1` succeeds against an empty tree and yields

```
a
  b
    c
      1
```

An index or a selector in a target does **not** create arbitrarily:

* `[k]` selects the existing child at position `k`. If `k` equals the number of children, one is appended. If `k` is greater, it is an error: an implementation MUST NOT pad the child list with empty or absent nodes.
* `{n}` selects the n-th existing match, and is an error if there are fewer.
* A group `.(expr)` is evaluated to a name and then behaves as a name element.

**The write.** `=` evaluates its right operand, then writes it into the target. What "into" means depends on how the target was addressed, and mirrors the asymmetry paths already have (path RFC 6.3):

* A target addressed by a **name element** or a selector names its content, so the value replaces the node's **children**. It is read back with `String()`.
* A target addressed by an **index** is an item, so the value replaces the **node itself**, content and children. It is read back with `ThisString()`.

So `a.b = 'd'` gives `b` a child holding `d`, while `list[1] = 99` makes the second item `99` rather than giving it a child.

For a name element: 

```
a                before
  b c

a.b = 'd'

a                after
  b d
```

The value becomes the target's single child, because that is how a document holds content: the node names it, the child holds it (core specification, section 2.2). If the value is a node, that node's children replace the target's children, and its own content is discarded — the target already has a name.

**Compound assignment.** `x op= y` is `x = x op y`, with `x` evaluated once. The target MUST already exist and hold a value; a compound assignment to a missing or empty target is an error. Every rule of section 6.2 applies to the operation, so `/= 0` is an error and `%=` requires integers.

**The result** of an assignment expression is the value assigned. This is what makes `a = b = 1` work.

### 6.6 Order of Evaluation

Operands are evaluated left to right, subject to short-circuiting. Where an operand contains a call or an assignment, this order is observable, so implementations MUST NOT reorder.

For an assignment, the right operand is evaluated before the target is resolved or created. A failure in the right operand therefore leaves the tree untouched.

An assignment is **all or nothing**: if the target cannot be resolved, any nodes created while trying MUST be removed. A rejected assignment MUST NOT leave behind the prefix of a path that turned out to be unusable, as `t.new{5} = 1` otherwise would.



## 7. Effects

Evaluation modifies the tree only through assignment (section 6.5) and through calls: a host function may do anything, including modifying the tree it was reached through.

An implementation MUST NOT cache, reorder or elide either. In particular, an expression is not safe to evaluate twice unless it is known to contain neither.

An expression from an untrusted source can therefore rewrite the tree it is evaluated against. A host that evaluates untrusted input SHOULD evaluate it against a copy, and MUST NOT assume that a tree survives evaluation unchanged. See also section 10.



## 8. Host Values and Calls

This section defines the interface between expressions and the host program. It is the reason the layer exists.

### 8.1 Host Values

A node's content may be any value the host places there — a function, a struct, a pointer, a map. Such a value is a **host value**. A parser never produces one.

### 8.2 Member Resolution

When a path element with name `m` is applied to a value `v`, the following are tried **in order**, and the first that succeeds wins:

1. If `v` is a node, a child of `v` whose name is `m` (path RFC section 6.2).
2. If `v` is, or simplifies to, a host value: a **method** named `m` on it.
3. An exported **struct field** named `m`, on it or on what it points to.
4. If it is a map with string-compatible keys, the entry under key `m`.
5. If `v` is a node and `m` is a built-in name, the built-in (section 8.4).

If none succeeds, the path matches nothing and evaluates to nil (section 6.1).

The tree therefore shadows the host value: a child node named `m` hides a method named `m`. Implementations MUST preserve this order, so that adding a method to a host type cannot change the meaning of an expression that already resolved through the tree. For the same reason the built-ins come last: a document that happens to contain a node named `_len` keeps its own meaning.

Unexported methods and fields MUST NOT be reachable.

### 8.3 Calls

A call applies to the value the preceding path element selected, after simplification (section 2.2). That value MUST be a function, or a method obtained by section 8.2; otherwise the call is an error.

Arguments are evaluated left to right, each as a full expression, before the call is made.

**Arity** MUST be checked before the call: a call with the wrong number of arguments is an error and MUST NOT be attempted. Variadic functions accept any number of arguments from the fixed count upwards.

**Argument conversion.** Each argument is converted to the parameter's type:

* If the value's type is identical or assignable to the parameter type, it is used unchanged.
* A number is converted between integer and float kinds if the value is representable.
* A string is converted to a number if it parses as one (section 2.3), and to a boolean if it is `"true"` or `"false"`.
* A node is converted by simplification (section 2.2), then by the rules above; a parameter of node type receives the node itself.
* nil is passed as the zero value of the parameter type.
* Otherwise the call is an error.

An implementation MUST NOT substitute a zero value for an argument it could not convert. Silently calling with a zero is a defect, not a fallback.

**Results.**

| Signature | Result |
|---|---|
| no results | nil |
| one result | that value |
| `(T, error)` | `T`, or the error if it is non-nil |
| anything else | an error |

**Panics.** A panic in a host function MUST be recovered and reported as an error. It MUST NOT propagate to the caller of the evaluator, which would let a data file crash its host.



### 8.4 Built-in Members

A small set of names computes something about a node rather than addressing part of it. They are applied to a node, never to a host value, and are resolved last (section 8.2).

| Name | Value | Kind |
|---|---|---|
| `_this` | the node itself, unsimplified | node |
| `_len` | the number of children | integer |
| `_string` | the text below the node: the textual form of its first child | string |
| `_name` | the node's own text | string |
| `_thisString` | the same as `_name`, under the reference's name | string |
| `_raw` | what a template substitution would write for the node (template RFC section 4), marked as safe HTML | host value |

`_this` is the identity, and exists so that the whole context, or a node that would otherwise simplify, can be passed to a call: `render(_this)`, `count(items._this)`.

`_string` and `_name` name the same distinction the data model draws: given `port 8080`, the node `port` has `_string` of `8080` and `_name` of `port`. `_name` is how a template reads the name of a loop element (template RFC section 5.2); `_thisString` is kept for templates written against the reference.

`_raw` exists for the HTML escaping mode of the template RFC (section 4.1): `$page.body._raw` writes the body unescaped. Its value is a host value of the implementation's safe-HTML type, so it means nothing different outside that mode, where it renders as the same text a plain substitution would. It marks a value as safe because the *expression* asks for it, which is to say the template author: a document cannot produce it, since a node named `_raw` shadows the built-in (section 8.2) and yields the document's own, unmarked, content.

These names are chosen to begin with an underscore because that prefix is rare in documents; it is not reserved. A node named `_len` is perfectly legal and takes precedence over the built-in, per section 8.2.

Built-ins belong to this layer only. An implementation MUST NOT apply them when evaluating a **strict** path (path RFC), because a path there addresses nodes and nothing else: `_len` computes a value rather than naming a location. An addressing API and an expression therefore differ on `a._len`, deliberately.

A path may continue through `_this`, since its value is a node. It may not usefully continue through the others, whose values are not nodes; doing so matches nothing and yields nil.

## 9. Errors

An implementation MUST report the following:

* A malformed expression: an unknown character, an unterminated string or index, a missing `)`, a missing operand, a trailing comma in an argument list, a bare `&` or `|`, an integer literal that does not fit in 64 bits.
* A malformed path, per path RFC section 8.
* Division or modulo by zero; `%` on a float.
* Unary `-` on a non-number.
* An ordering comparison of booleans.
* A call of something that is not callable; wrong arity; an argument that cannot be converted; an unsupported result signature; a recovered panic.
* An assignment whose target is not a path, or is a set; an index target more than one past the end; a selector target that does not exist; a compound assignment to a target that does not exist or holds nothing (section 6.5).
* Exceeding a documented limit (section 10).

A parse error SHOULD identify the character offset within the expression.

An empty path result is **not** an error (section 6.1). Implementations MUST distinguish "this expression is wrong" from "this tree does not contain that".



## 10. Limits

Implementations MUST document:

* the maximum nesting depth of an expression, which bounds parser recursion;
* the maximum call depth, since a host function may re-enter the evaluator;
* any bound on evaluation time or result size.



## 11. Extensibility

Assignment, which earlier drafts of this document reserved, is now specified (sections 3.3, 6.5). Level 0 is spent.

What remains reserved is small: a **single** `&` or `|`, which is an error today (section 3.3) and is held for a possible bitwise layer — `&`, `|`, `^`, `<<`, `>>`. Adding those later cannot change the meaning of any conforming expression, because no conforming expression may contain a lone `&` or `|` now.

Nothing else is open. The operator set of section 3.3 is otherwise closed, and the token class of path RFC section 4.2 is fully committed: a future operator would have to be built from characters already reserved here.



## 12. Differences from the Reference Implementation

Each of these is deliberate. `path-notes.md` records the same exercise for paths.

| Behaviour | `rveen/ogdl` | Here |
|---|---|---|
| Text that looks like a number, in arithmetic | depends on how the document was parsed: `FromString` keeps text, `FromStringTypes` converts | always a number, since a stream carries only text (section 2.3). `'4'+3` is `7`, not `"43"` |
| Comparison kind | the first operand decides; a non-numeric second operand makes the comparison false | symmetric (section 6.3) |
| Ordering comparison of strings | always false | lexicographic (section 6.3) |
| Division by zero | evaluates to nil | an error (section 6.2) |
| Argument that cannot be converted | passed as the zero value | an error (section 8.3) |
| Result signature other than the three in 8.3 | first result taken | an error |
| Operator recognition | inferred from the first character and the length of a name | lexed (section 3.3), so a name beginning with an operator character is never mistaken for one |
| Expression representation | an OGDL tree with sentinel names `!e`, `!p`, `!i`, `!a`, `!g` | a typed syntax tree; sentinel names never enter the data model |
| `true` and `false` written bare | evaluate to nil — they are parsed as paths and match nothing | boolean literals (section 3.4); quote to address a node so named |
| A nil operand in arithmetic | concatenated as an empty string, so `missing+1` is `"1"` | an error (section 6.2) |
| Unary `-` on a non-number | yields the operand's text, so `-s` is `"s"` | an error (section 4.2) |
| Compound assignment to a missing target | invents a result: `missing += 1` assigns `1`, `missing *= 2` assigns `0`, `missing /= 2` assigns the string `"infinity"`, `%=` assigns `"undefined"` | an error (section 6.5) |
| Index target past the end | pads the child list with placeholder nodes: `list[9] = 1` on a three-item list leaves six empty children that serialize as `_` | an error, except for the single position that appends (section 6.5) |
| Assignment associativity | left, following the generic precedence loop. `a.b = b = 1` writes the *parse tree* into the data: it grafts nodes named `!p` into the document | right, so `a = b = 1` assigns 1 to both (section 4.1) |
| Compound assignment on text | `a.b += 1` where `b` holds `1` yields `11`, since `+` concatenates unparsed text | `2`, following the coercion rule of section 2.3 |

Two behaviours of the reference are defects rather than design, and were verified by running it:

* `!(expr)` ignores its operand: `!(2>1)` and `!(a.b==1)` both evaluate to `true`, although `!true` correctly gives `false`. Negation of a parenthesized expression is simply wrong.
* An out-of-range index yields a node containing a typed nil rather than nothing — the same defect `path-notes.md` records for paths.

Neither is reproduced here.

The first row is the one to watch when porting. Our data model has no way to distinguish a node that was quoted in the source from one that was not — quoting conveys no type (core specification, section 2.4) — so the reference's typed/untyped distinction cannot be reproduced, and the useful reading was chosen instead.



## Appendix A. Test Vectors

Evaluated against this context:

```
a
  b 1
  c 2
n 4
s abc
t
  x 1
  x 2
list
  10
  20
  30
ptr b
```

### A.1 Literals and Arithmetic

| Expression | Value | Kind |
|---|---|---|
| `1` | 1 | integer |
| `1.5` | 1.5 | float |
| `1e3` | 1000 | float |
| `'abc'` | abc | string |
| `true` | true | boolean |
| `1+2*3` | 7 | integer |
| `(1+2)*3` | 9 | integer |
| `1-2+3` | 2 | integer |
| `7/2` | 3 | integer |
| `7.0/2` | 3.5 | float |
| `7%2` | 1 | integer |
| `-3+1` | -2 | integer |
| `'ab'+'cd'` | abcd | string |
| `'ab'+1` | ab1 | string |

### A.2 Paths as Operands

| Expression | Value |
|---|---|
| `a.b` | 1 |
| `a.b+a.c` | 3 |
| `n+1` | 5 |
| `s` | abc |
| `s+'d'` | abcd |
| `list[1]` | 20 |
| `list[1+1]` | 30 |
| `t.x{1}` | 2 |
| `a.(ptr)` | 1 |
| `missing` | nil |
| `missing+1` | an error: nil operand |

### A.3 Comparison and Logic

| Expression | Value |
|---|---|
| `2>1` | true |
| `2>=2` | true |
| `n==4` | true |
| `s=='abc'` | true |
| `s<'abd'` | true |
| `a.b==1 && a.c==2` | true |
| `a.b==9 \|\| a.c==2` | true |
| `!(a.b==1)` | false |
| `a.b==9 && missing.x()` | false, and the call is not made |

### A.3b Assignment

Each row starts from the context above, and shows the canonical form of the part of the tree that changed.

| Expression | Result | Tree after |
|---|---|---|
| `a.b = 9` | 9 | `a` → `b` → `9` |
| `a.b = 1+2` | 3 | `a` → `b` → `3` |
| `x.y.z = 1` | 1 | `x` → `y` → `z` → `1` created |
| `a.b += 1` | 2 | `a` → `b` → `2` |
| `a.b *= 3` | 3 | `a` → `b` → `3` |
| `n = n + 1` | 5 | `n` → `5` |
| `a.b = a.c` | 2 | `a` → `b` → `2` |
| `p = q = 7` | 7 | `p` → `7` and `q` → `7` |
| `list[1] = 99` | 99 | the second child of `list` becomes `99` |
| `list[3] = 99` | 99 | appended, since 3 is the child count |
| `t.x{1} = 9` | 9 | the second `x` becomes `9` |

Errors:

| Expression | Reason |
|---|---|
| `1 = 2` | target is not a path |
| `1+1 = 2` | target is not a path |
| `f() = 2` | target is the result of a call |
| `t.x{} = 2` | target is a set |
| `list[9] = 1` | index more than one past the end |
| `t.x{5} = 1` | selector target does not exist |
| `missing += 1` | compound assignment to a missing target |
| `a.b /= 0` | division by zero; the tree is unchanged |

### A.4 Errors

| Expression | Reason |
|---|---|
| `1/0` | division by zero |
| `1%0` | modulo by zero |
| `1.5%2` | modulo on a float |
| `-s` | unary minus on a non-number |
| `true<false` | ordering comparison of booleans |
| `1+` | missing operand |
| `(1+2` | missing `)` |
| `a=1` | bare `=`, reserved for assignment |
| `f(1,)` | trailing comma |
| `'abc` | unterminated string |

### A.5 Host Values

With `fn` holding `func(int) int`, `obj` holding a pointer to a struct with method `Add(int) int`, exported field `Name`, and method `Fail() (int, error)` returning a non-nil error:

| Expression | Result |
|---|---|
| `fn(2)` | whatever `fn` returns for 2 |
| `obj.Add(2+3)` | the method result |
| `obj.Name` | the field value |
| `obj.Add(1,2)` | an error: arity |
| `obj.Missing` | nil |
| `obj.Fail()` | the returned error |
| `obj.Name(1)` | an error: not callable |
