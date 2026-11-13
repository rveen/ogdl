# RFC: OGDL Templates

## Status of This Memo

This document specifies the OGDL template language, in the style of an IETF RFC.

It depends on `ogdl-rfc.md` (the "core specification"), `ogdl-path-rfc.md` (the "path RFC") and `ogdl-expr-rfc.md` (the "expression RFC").

No prior specification of this layer exists. The reference implementation, `github.com/rveen/ogdl`, has carried a template engine for years; this document specifies it, and departs from it where its behaviour was accidental rather than designed. Section 10 lists every departure.

Distribution of this memo is unlimited.

The key words MUST, MUST NOT, REQUIRED, SHALL, SHALL NOT, SHOULD, SHOULD NOT, RECOMMENDED, MAY, and OPTIONAL in this document are to be interpreted as described in RFC 2119.



## Abstract

An OGDL template is a text of any kind — plain text, HTML, XML, OGDL — in which `$` switches from text to the expression language. Processing a template against a context node substitutes values, evaluates conditionals and loops, and produces text. This document defines the syntax, the processing model, the output rules, variable scoping, and the error model.



## 1. Introduction

A template is text with holes:

```
Hello, $user.name

$if(user.admin)
  You have admin rights.
$end

$for(item, cart)
  - $item.name: $item.price
$end
```

`$` is the only metacharacter. Everything else is copied to the output unchanged, so a template can be written in any format without escaping that format's own syntax.

Processing takes a template and a **context node**, and produces text. The expression RFC defines what may appear after `$`; this document defines only where those expressions may appear, and what is done with their values.



## 2. Syntax

### 2.1 Overall

```
template  = ( text | substitution | silent | conditional | loop | break | escape )*
text      = any run of characters containing no unescaped '$'
```

### 2.2 Substitution

```
substitution = "$" path | "$" "{" path "}"
```

`$path` evaluates an extended path (expression RFC section 5) and writes its value to the output (section 4).

The braced form exists to end the path where text resumes: `${a}bc` substitutes `a` and then writes `bc`, where `$abc` would look up `abc`. Spaces are permitted inside the braces.

Because the path is the extended one, a substitution may call: `$user.FullName()`, `$fmt(price)`.

### 2.3 Silent Evaluation

```
silent = "$" "(" expression ")"
```

`$(expr)` evaluates a full expression and writes **nothing**. It exists for effects — chiefly assignment:

```
$(total = price * quantity)The total is $total.
```

An expression whose value is discarded is not an error.

### 2.4 Conditional

```
conditional = "$if" "(" expression ")" template
              ( "$elseif" "(" expression ")" template )*
              ( "$else" template )?
              "$end"
```

The expression is coerced to a boolean by the total rule of expression RFC section 2.3, so a missing path is false and any non-empty value is true.

At most one branch is processed. `$else` and `$elseif` bind to the nearest unclosed `$if`.

### 2.5 Loop

```
loop = "$for" "(" name "," expression ")" template "$end"
break = "$break"
```

`name` names the loop variable. It is a path token (path RFC section 4.2) that begins with a letter or an underscore. Digits are path tokens, so this rule is what makes `$for(1, b)` an error rather than a loop over a variable named `1`; it is the same rule that governs what follows a `$` in section 2.6.

The expression is evaluated once, before the loop. Its value is iterated as defined in section 5. For each iteration the loop variable is bound, and the body is processed.

`$break` ends the innermost enclosing loop immediately. A `$break` outside any loop is an error.

### 2.6 Escape and Literal `$`

```
escape = "$$"
```

`$$` writes a single `$`.

A `$` that is not followed by `$`, `(`, `{`, a letter or `_` is **literal**: it is written to the output as it stands. This makes `$5.00`, `US$`, and `a $ b` safe without escaping, which matters because templates are usually written by people who are not thinking about this specification.

A `$` followed by a letter or `_` is always a substitution or a directive; to write one literally, use `$$`.

### 2.7 Directive Names

The words `if`, `elseif`, `else`, `end`, `for` and `break`, when they appear as the whole path immediately after `$`, are directives and not substitutions. To substitute a node with one of those names, use the braced form and quoting: `${'if'}`.

Directives are recognised only at the start of a substitution, so `$a.for` and `$forest` are ordinary paths.



## 3. Processing Model

Processing walks the template in order, writing to an output text.

* Text is written unchanged, byte for byte. A template MUST NOT normalize line endings, trim whitespace, or reformat what it copies.
* A substitution writes the rendering of its value (section 4).
* A silent evaluation writes nothing.
* A conditional processes at most one branch.
* A loop processes its body once per iteration (section 5).

Evaluation order is the order of the template. Because expressions may assign and may call (expression RFC section 7), this order is observable, so implementations MUST NOT reorder, cache or elide any part of it.

The context node is the same node throughout. Loop variables (section 5.2) are bound beside it, never in it. A template MAY modify the context, through `$(...)` assignment or a call; that is the mechanism section 2.3 exists for.



## 4. Rendering a Value

A substitution renders its value as follows.

| Value | Rendering |
|---|---|
| nil | the empty string |
| a value that simplifies to a scalar (expression RFC 2.2) | its textual form (expression RFC 2.3) |
| any other node | the canonical serialization of the node's **children**, with the final line break removed |
| a host value | its textual form |

A path that matches nothing therefore renders as nothing, and is not an error: templates routinely reference optional data.

The third row is what makes a subtree printable:

```
b
  1
  2
```

`$b` renders `1\n2`, because `b` has more than one child and so does not simplify. This is the same rule as `$b` rendering `1` when `b` has a single leaf child — in both cases what is written is what lies *below* `b`, never `b`'s own name.

Rendering uses canonical serialization (core specification section 8), so a name that requires quoting is quoted. Implementations MUST NOT invent a second, unquoted tree format for templates.

### 4.1 HTML Escaping

An implementation MAY offer an **HTML escaping mode**, chosen by the host when it processes a template. The template text does not select it, so the same template means the same thing in either mode except at substitutions.

In this mode, the rendering of every substitution (section 4) is escaped before it is written: `&`, `<`, `>`, `"` and `'` are replaced by character references. Nothing else changes:

* Template text is still written unchanged (section 3). Markup belongs in the text; data belongs in substitutions.
* A silent evaluation writes nothing, so there is nothing to escape.
* A value **marked as safe HTML** is written unescaped. Two parties can mark a value, and both are code. The **template** marks one with the built-in `_raw` (expression RFC section 8.4): `$page.body._raw` writes the body as it stands. The **host** marks one through its interface; in Go, by giving it a dedicated string type, for instance as the result of a sanitizing function. A **document** cannot: a stream carries no types (core specification section 2.2), and a node named `_raw` shadows the built-in rather than invoking it.

The escaping is not contextual. It makes a value safe as element content and inside a **quoted** attribute value. It does not make a value safe inside `<script>` or `<style>`, in an unquoted attribute, or as a URL (`href="$link"` admits `javascript:`). A template that places data in those positions MUST escape or validate it by other means.



## 5. Loops

### 5.1 What Is Iterated

The loop expression is evaluated once. Then:

* If the value is a node, the iteration is over its **children**, in order.
* If the value is nil, the loop body is not processed at all. This is not an error.
* If the value is a host slice or array, the iteration is over its elements.
* Any other value is an error.

A loop expression that is a path is taken *before* operand simplification (expression RFC 2.2). A node with a single leaf child therefore iterates that one child, and a node with no children iterates nothing; simplified, they would be a string, which is not iterable. The exception is a node whose single leaf child holds a host value, the form a host value takes in the tree: that value is iterated as above, so a host slice under `list` iterates its elements and a host map is an error.

The sequence to be iterated is fixed before the first iteration. If the body modifies the tree being iterated, the iteration is unaffected.

### 5.2 The Loop Variable

For each iteration, the loop variable is bound to the iterated element itself. So for

```
cart
  item
    name Widget
  item
    name Gadget
```

`$for(x, cart)` binds `x` to each `item` in turn, and `$x.name` renders `Widget`, then `Gadget`.

What `$x` renders is what any path to that element renders (section 4): a leaf element, such as each child of `b`, simplifies to its own text, so `$for(x, b)[$x]$end` renders `[1][2]`. The element's own name is always `x._name` (expression RFC section 8.4), whatever the element holds. An element that is a host value (section 5.1) is bound as that value, so `$x.Name` reaches its field.

**Resolution.** A variable is consulted only for the **first** element of a path, and before the children of the context: `x.name` starts at the bound element, while `user.x` is the child `x` of `user` as usual. A selector after the variable addresses the element itself, which is occurrence 0 and the only member of its set: `x{0}` is the element, `x{1}` matches nothing, and `x{}` is a set of one.

**Assignment.** Because the variable is the element, assigning through it writes into the element, and so into the tree that holds it: `$(x.price = 0)` changes the price of the current `item`. Assigning to the variable itself, `$(x = 1)`, replaces the element's children, as an assignment to any name element does (expression RFC section 6.5).

**Scoping.** The binding is confined to the loop, and is never written into the context:

* A node of the context with the same name is hidden from paths in the body for the duration of the loop, and is untouched. It remains reachable as `_this.x`.
* An enclosing loop's variable of the same name is hidden likewise, and bound again when the inner loop ends.
* The binding ends with the loop, including when the loop ends through `$break` or an error.

Binding a variable therefore never modifies the context, and nested loops with the same variable name behave as expected.

### 5.3 Break

`$break` stops the innermost enclosing loop. Text already written stays written; the loop variable is restored as in section 5.2.



## 6. Errors

Processing reports an error for:

* A malformed template: an unterminated `$(`, `${`, `$if(` or `$for(`; a missing `$end`; an `$end`, `$else` or `$elseif` with no open `$if`; an `$else` after an `$else`; a `$break` outside a loop; a `$for` whose first argument is not a single token.
* Any error the expression RFC defines, raised while evaluating a substitution, condition, loop expression or silent expression.
* Iterating a value that is not iterable (section 5.1).

A parse error SHOULD identify the offset in the template; a processing error SHOULD identify the offset of the expression that raised it.

A path that matches nothing is **not** an error, at any of these positions. This is the same distinction the path RFC draws, and it is what allows a template to reference optional data without guarding every use.

Implementations MUST NOT swallow errors. Producing a partial or empty output where an expression failed hides the fault at exactly the moment it matters.



## 7. Limits

Implementations MUST document:

* the maximum nesting depth of conditionals and loops;
* the maximum size of the output, or the absence of such a limit;
* the expression limits of expression RFC section 10, which apply unchanged.

A loop over a large tree, inside a loop, is the obvious way to make a small template produce an unbounded output. An implementation processing untrusted templates SHOULD bound the output.



## 8. Security Considerations

A template is code. Through `$(...)` it can assign into the context, and through the expression layer it can call any function or method the host placed in the tree (expression RFC section 8).

Therefore:

* A template from an untrusted source MUST be treated as untrusted code. Processing it against a context that holds host functions gives it those functions.
* A host processing untrusted templates SHOULD process them against a context prepared for that purpose, containing only data, and SHOULD copy the context if it must not be modified.
* Template output is text. Apart from the optional HTML mode of section 4.1, this document defines no escaping. A template that interpolates untrusted data into HTML without that mode, or into SQL or a shell command, is vulnerable in the ordinary way. A host generating HTML SHOULD use the HTML mode; for other formats, escaping is the host's responsibility, and is best done by exposing an escaping function in the context.
* Marking a value as safe HTML (section 4.1) disables the protection for that value. A host SHOULD mark only markup it produced or sanitized itself, and a template SHOULD apply `_raw` only to content whose source is trusted. An untrusted template can apply `_raw` to anything, which is one more reason it is untrusted code.



## 9. Conformance

A conforming implementation:

* MUST process every well-formed template as sections 3 to 5 define.
* MUST reject every malformed template, and MUST report rather than swallow evaluation errors (section 6).
* MUST NOT modify the context to bind a loop variable (section 5.2).
* MUST copy text unchanged (section 3).
* MUST document its limits (section 7).
* MAY offer the HTML escaping mode; if it does, it MUST escape as section 4.1 defines and MUST NOT escape when the mode is not chosen.
* MUST reproduce appendix A.



## 10. Differences from the Reference Implementation

All of the following were verified by running the reference.

| Behaviour | `rveen/ogdl` | Here |
|---|---|---|
| Loop variable scope | **destroys data**: any existing node of that name is deleted before the loop and the binding is left behind after it. With `n 4` in the context, `$for(n, b)$n$end$n` renders `122` and leaves `n` holding `2` — the original 4 is gone for good | scoped to the loop and never written into the context (section 5.2). The same template renders `124` |
| What the loop variable is | a node named after the variable whose single child is the element, so templates step into it: `$x[0].name`, and `$x[0]` for the element's own text | the element itself: `$x.name`, and `$x._name` for its own name (section 5.2). A template is migrated by rewriting, inside the loop body, `x[0]` followed by a path element to `x` and any other `x[0]` to `x._name`; the reference's `$for(x, b)[$x[0]]$end` and this `$for(x, b)[$x._name]$end` both render `[1][2]` |
| Errors | discarded. `$(1/0)` renders as nothing; `$if(n>1)x` with no `$end` renders the whole template as nothing | reported (section 6) |
| A `$` before a non-path character | swallows the `$` *and what follows*: `$5.00` renders as the empty string, and `a $ b` as `a  b` | literal, so both render unchanged (section 2.6) |
| Escaping a `$` | `$\` | `$$`, with `$\` no longer special (section 2.6) |
| A directive followed by `[` | the directive name is parsed as a path, so the bracket is taken as an index: `$end[$x]` swallows the text `[$x]` | `$end` is a directive and `[$x]` is text and a substitution |
| `$path[i]` rendering | renders the indexed node *including its own name*, because an index wraps its result | renders what lies below it, like every other substitution (section 4) |
| `$if` / `$else` chaining | tracked by a flag shared across the whole template level, so `$else x$end` with no `$if` silently renders ` x` | a structural error (section 6) |
| Iterating a non-iterable | silently skipped | an error (section 5.1) |

The first two rows matter most. The loop variable does not merely leak — it deletes whatever it shadows, so a template can quietly destroy part of the context it was given. And discarding errors turns a typo into silently missing output, the failure mode hardest to notice in generated text: a template with an unclosed `$if` produces nothing at all, with no indication why.



## Appendix A. Test Vectors

Context:

```
user
  name Jenny
  admin true
b
  1
  2
cart
  item
    name Widget
    price 10
  item
    name Gadget
    price 20
n 4
```

### A.1 Substitution

| Template | Output |
|---|---|
| `Hello, $user.name` | `Hello, Jenny` |
| `${user.name}!` | `Jenny!` |
| `$b` | `1\n2` |
| `$missing` | `` |
| `[$missing]` | `[]` |
| `$n` | `4` |
| `$$5.00` | `$5.00` |
| `$5.00` | `$5.00` |
| `a $ b` | `a $ b` |

### A.2 Silent Evaluation

| Template | Output |
|---|---|
| `$(x = 1)$x` | `1` |
| `$(x = n * 2)$x` | `8` |
| `$(x = 1)` | `` |

### A.3 Conditionals

| Template | Output |
|---|---|
| `$if(user.admin)yes$end` | `yes` |
| `$if(missing)yes$end` | `` |
| `$if(missing)yes$else no$end` | ` no` |
| `$if(n>10)a$elseif(n>3)b$else c$end` | `b` |
| `$if(n>10)a$elseif(n>100)b$else c$end` | ` c` |
| `$if(n>1)$if(n>3)deep$end$end` | `deep` |

### A.4 Loops

| Template | Output |
|---|---|
| `$for(x, b)[$x]$end` | `[1][2]` |
| `$for(x, cart)$x.name $end` | `Widget Gadget ` |
| `$for(x, cart)$x._name $end` | `item item ` |
| `$for(x, missing)never$end` | `` |
| `$for(x, user.name)[$x]$end` | `[Jenny]` |
| `$for(x, b[0])[$x]$end` | `` |
| `$for(x, b)$x$break$end` | `1` |
| `$for(x, b)$for(y, b)$x$y$end$end` | `11122122` |

The nested loop emits `11`, `12`, `21`, `22` in turn: the inner loop rebinds `y` on each pass while `x` stays bound, and the inner binding is gone before the outer loop advances (section 5.2).

`user.name` has a single leaf child and `b[0]` has none; neither is simplified before the loop (section 5.1).

### A.5 Scoping

| Template | Output | Note |
|---|---|---|
| `$for(n, b)$n$end$n` | `124` | `n` is restored to `4` after the loop |
| `$for(x, b)$x$end$x` | `12` | `x` did not exist before, and does not after |
| `$for(n, b)[$n $_this.n]$end` | `[1 4][2 4]` | the shadowed `n` is untouched during the loop |

### A.6 Errors

| Template | Reason |
|---|---|
| `$if(n>1)x` | missing `$end` |
| `$end` | `$end` with no `$if` or `$for` |
| `$else x$end` | `$else` with no `$if` |
| `$break` | `$break` outside a loop |
| `$for(n)x$end` | `$for` needs a variable and an expression |
| `$for(1, b)x$end` | the loop variable is not a token |
| `$(1/0)` | division by zero |
| `$user.name(` | unterminated call |
| `${user.name` | unterminated brace |

### A.7 HTML Escaping

Processed in the HTML escaping mode of section 4.1, with the context above.

| Template | Output |
|---|---|
| `<p>$user.name</p>` | `<p>Jenny</p>` |
| `$(x = '<b>')$x` | `&lt;b&gt;` |
| `$(x = 'a&b')$x` | `a&amp;b` |
| `$(x = 'a"b')<a title="$x">` | `<a title="a&#34;b">` |
| `$b` | `1` LF `2` |
| `$(x = '<b>')$x._raw` | `<b>` |

The same templates processed without the mode render `<b>`, `a&b` and `<a title="a"b">` for the second to fourth rows.
