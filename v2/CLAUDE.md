# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a single test
go test -v -run TestName ./...

# Run the character classification utility (not a test)
go run test/lexer/main.go
```

This is a Go module (`github.com/rveen/ogdl/v2`, `go 1.21`) with no dependencies outside the standard library; `ogdlrf/` is a subpackage.

## Architecture

OGDL is a Go implementation of the Ordered Graph Data Language, a human-readable indentation-sensitive format representing ordered trees. The implementation is a single package (`github.com/rveen/ogdl/v2`) organized as a three-stage pipeline:

**Lexer → Parser → Graph**

### graph.go — Node representation
`Node` has `Data interface{}` and `Out []*Node` children. The field is *not* called Name on purpose: the RFC's "name" is the textual form a node takes in a stream, whereas `Data` may hold any Go value (int, bool, []byte, *Node, …). Parsing always yields `string` data — a stream carries no types; other types come from programmatic construction or a later evaluation stage.

`nameString()` resolves `Data` to its stream text and is the single point where that mapping lives; comparison (`S`), `ThisString`, `String` and serialization all go through it. Navigation: `N(i)` by index, `S(s)` by text; both return an empty node (not nil) when nothing matches.

There is no `Graph` type or alias — the break with `github.com/rveen/ogdl` naming is deliberate.

Serialization is canonical (RFC 8): two spaces per level, one node per line, LF breaks, minimal double quoting, only `\\ \" \n \t` escaped. `Canonical()` returns `(string, error)`; `Text()` is the convenience form that drops the error. `=` is ordinary content, so `a=b` is written unquoted. `Compact()` (document.go) is the compact form of RFC 8.5 — leaf children on the head's line — which reparses to the same tree but is never used for equality. The receiver is treated as the root and is not itself emitted. A name containing a control character other than TAB/LF/CR cannot be serialized and yields `ErrControlCharacter`.

### lexer.go — Tokenization
`Lexer` reads from `io.Reader` and emits `Token` values carrying `Line`/`Col`. Token types: `TokenIndent`, `TokenSpace`, `TokenWord`, `TokenQuoted`, `TokenBlock`, `TokenComment`, `TokenBreak`, `TokenEOF`. Key behaviors:
- `TokenIndent` is emitted only for a line's leading whitespace, so the parser never guesses whether spaces are indentation
- LF, CR and CRLF are equivalent; a leading U+FEFF is skipped
- `readRune` rejects invalid UTF-8 and control characters other than TAB/LF/CR, so no other code path repeats the check
- Comments follow RFC 4.6: `#` at line start or after a space, followed by a space, break or EOS
- Quoted strings may span lines (break → LF, following indentation dropped); undefined escapes are errors, not literals
- A word of exactly `\` becomes `TokenBlock`, never a name
- There are no pairs: `=` is ordinary content and never splits a word. Pairs (`key=value`) existed in an intermediate draft and were withdrawn (`rfc/analysis.md` Part 5) because they made one element denote two nodes and gave a single line a third level — do not reintroduce them or any other in-line nesting marker
- `pending` holds pushed-back input, which lets `ReadBlock` return the line that terminated a block

### parser.go — Tree construction
`Parse()` returns an unnamed root `Node` whose children are the depth-zero nodes. A line is a **head followed by arguments** (RFC 6.5): `a b c` is `a` with children `b` and `c` — siblings, *not* a chain (earlier drafts chained; that rule is gone, see `rfc/analysis.md` Part 4). Only the head is pushed on the `(node, depth)` stack, so an indented line attaches to the head; arguments and blocks are leaves. The root sits at depth -1 and depth 0 is always open.

`ParseString(s)` and `Parse(r)` are the package-level shortcuts.

### document.go — comments and layout
`ParseDocument` returns a `*Document`: the tree plus `Comments map[*Node]*Comments` (`Before`, `Line`, `After`; entries are `# text` or `""` for a blank line) and an unexported record of which nodes were written inline. Attachment follows RFC Appendix B and is done in the parser (`collect`, `attachEnd`, `attachLine`, `attachEOF`) only when `Parser.doc` is set — a plain `Parse` never pays for it. `Format()` writes compact form, keeping each parsed node inline or on its own line as written; a node with comments is never inline. Presentation is keyed by node pointer, so nodes replaced by `Set` lose their comments. Must not regress: a normalized document round-trips byte for byte (`TestDocumentRoundTrip`), and retained comments never change the tree.

### path.go — the Get API
`Get(path)` evaluates an OGDL path and returns a node, never nil (empty node when nothing matches or the path is malformed); `GetErr` reports malformed paths as `*PathError`. `Path(s)` renders a parsed path for debugging.

Element semantics: a token selects the first child whose *text* matches; `[n]` the n-th child, zero-based; `{n}` the n-th child matching the preceding token; `{}` all of them, returned as a **set** — a synthetic node whose children are the matches. A token after a set maps over its members. A token is letters, digits and `_` only — deliberately narrow, so every operator character stays free for the expression layer (`Eval`, below). Anything else MUST be quoted: `host.'192.168.0.10'`, `iface.'eth0-1'`. Do not widen this class; `a-b` must stay a parse error, or it would be one token in a path and a subtraction in an expression.

The spec is `rfc/ogdl-path-rfc.md`; `rfc/path-notes.md` is the audit of the published ogdl.org path page that motivated it. This implementation deliberately diverges from `rveen/ogdl` on indexing and on `{}` — read the notes before changing path behavior.

### expr.go / eval.go / reflect.go / value.go — the expression layer
`ParseExpr(s)` produces a typed AST (`Literal`, `PathExpr`, `Unary`, `Binary`) — deliberately *not* the reference's OGDL-tree-with-`!e`-sentinels representation. `node.Eval(expr)` and `node.EvalString(s)` evaluate it against a context node, yielding `int64`, `float64`, `string`, `bool`, `*Node`, a host value, or nil.

`value.go` is the value model: `scalarOf` (operand simplification), `toNumber`/`toInt64`/`toFloat64`/`toString`/`toBool`, and the `Int64`/`Float64`/`Bool`/`Strings` accessors. Coercion happens *at the operator*, never in advance — a stream carries only text, so `"4"+3` is `7` here, unlike the reference.

`reflect.go` is the Go interop: `memberOf` (method → exported field → map key), `indexOf`, `callAny`, `convertArg`. Two rules that must not regress: the **tree shadows the host value** (a child node named `m` hides a method `m`), and an argument that cannot be converted is an **error** — never a substituted zero value.

Member resolution is unified in `evaluator.resolveMember` (eval.go), in the order of Expr RFC 8.2: child node → host member → **built-in** (`_this`, `_len`, `_name`, `_string`, `_thisString` — a synonym of `_name` —, `_raw`). Built-ins come last, so a document containing a node named `_len` keeps its own meaning, and they are gated by `evaluator.builtins` — **on** for `Eval` and templates, **off** for `resolvePath`, because a strict path addresses nodes and `_len` computes a value. `Get("a._len")` and `EvalString("a._len")` therefore differ, deliberately.

`set.go` is assignment: `Set(path, value)` / `SetErr` write into the tree, creating missing name elements. Assignment operators (`=`, `+=`, `-=`, `*=`, `/=`, `%=`) sit at precedence 0 and are the **only right-associative** operators, so `a = b = 1` nests rightwards.

Three rules that must not regress: the right operand is evaluated **before** the target is resolved, and target creation **rolls back** if resolution then fails, so a rejected assignment never leaves debris; a **name element** target replaces the node's children (read with `String()`) while an **index** target replaces the node itself (read with `ThisString()`), mirroring the same asymmetry in paths; and an index more than one past the end is an error rather than padding with placeholder nodes.

Evaluation is therefore no longer pure — `Eval` on untrusted input can rewrite the tree. Evaluate against a copy if that matters.

Spec: `rfc/ogdl-expr-rfc.md`, §12 of which lists every deliberate divergence from `rveen/ogdl` — including two verified reference bugs (`!(expr)` ignores its operand; bare `true`/`false` evaluate to nil).

### template.go — the template engine
`ParseTemplate(s)` → `*Template`; `(*Template).Process(ctx)` or `ProcessString(tpl, ctx)` render against a context node, returning `(string, error)`. Syntax: `$path`, `${path}`, `$(expr)` (silent, for assignment), `$if`/`$elseif`/`$else`/`$end`, `$for(name, expr)`/`$break`, `$$` for a literal `$`. A `$` not followed by `$`, `(`, `{`, a letter or `_` is literal, so `$5.00` and `a $ b` pass through. **HTML mode** (`ParseHTMLTemplate`, `ProcessHTMLString`, or `Template.EscapeHTML`) escapes every substitution with `html.EscapeString`, except a value of type `HTML` — produced by host code or by the built-in `_raw` (the node's rendering, unescaped); template text is never escaped. A document can never produce `HTML`: a node named `_raw` shadows the built-in, so its content stays escaped (tested). It is opt-in, so the plain mode and its test vectors are unchanged.

Non-negotiables, each with a test: **errors are reported, never swallowed** (a missing path still renders empty — that is not an error); the **loop variable is the element itself** (`$x.name`, `$x._name` for its own name), bound in `evaluator.vars` and never in the tree: it is consulted only for a path's first element, hides a context node of that name without touching it, and its binding ends with the loop, including on `$break` or error; and **text is copied byte for byte** (no line-ending or whitespace normalization).

Spec: `rfc/ogdl-template-rfc.md`, §10 of which records verified reference behavior — notably that its loop variable *deletes* the node it shadows, and that an unclosed `$if` renders the whole template as nothing.

### schema.go — the schema language
`ParseSchema`/`CompileSchema` → `*Schema`; `Validate(doc)` returns a `*ValidationError` listing every `Issue{Path, Message}`; `Apply(doc)` validates and then converts `!int`/`!float`/`!bool` matches to `int64`/`float64`/`bool`, leaving an invalid document untouched. Ill-formed schemas yield `*SchemaError` wrapping `ErrSchema`. A schema is an ordinary OGDL document: named particles (`port`, `rule*`), type particles (`!int`, `!string*`) and definitions (`!type action allow deny`, depth zero only). Matching (Schema RFC 5.1): named particles by name in any order, then type particles in order over the remaining children, **possessively, never backtracking**; the schema is closed. Paths in issues: a child whose name failed a type is addressed by index (`rule[0]`), every other child by name. Numbers use one decimal grammar (`isDecimal`), shared with `value.go`, so `!float` and arithmetic agree — no hex, `Inf` or `NaN`. Spec: `rfc/ogdl-schema-rfc.md`, Appendix A mirrored in `schema_test.go`.

### binary.go / log.go / ogdlrf — binary OGDL
Spec: `rfc/ogdl-binary-rfc.md`; Appendix A is mirrored by `TestBinaryVectors` and `TestBinaryIllFormed` — keep them in step. `MarshalBinary`/`Binary`/`WriteBinary` encode; `ParseBinary` decodes exactly one stream; `BinaryReader` reads concatenated streams, stopping at each stream's end (pass an `io.ByteReader`, or it reads ahead), and returns `io.EOF` only where a next stream would begin. A `[]byte` name is a data record and decodes back to `[]byte`; every other name is text and decodes to `string`. Must not regress: whatever `github.com/rveen/ogdl` v1 writes decodes to the same tree and re-encodes byte for byte (gstore's data is stored in this form); malformed input is a `*BinaryError` with the offset, never a partial tree; a data length is never allocated in advance. `Log` (log.go) is the append-only file of streams, addressed by position; the zero `Log` is in memory. `ogdlrf/` is the remote-function client and server, wire-compatible with v1 in both framings.

### Errors
All ill-formed input yields a `*SyntaxError{Line, Col, Err}` wrapping a sentinel (`ErrMixedIndentation`, `ErrInconsistentIndentation`, `ErrUnopenedDedent`, `ErrUnterminatedQuoted`, `ErrInvalidEscape`, `ErrControlCharacter`, `ErrInvalidEncoding`, `ErrBlockIndentation`, `ErrBlockNotLast`, `ErrNameTooLong`, `ErrTooDeep`); match with `errors.Is`. Rejection is the only mode — there is no lenient recovery. Limits are `Parser.MaxDepth` and `Lexer.MaxNameLength`.

### Indentation semantics
The indentation character (space or tab) is fixed by the first line with a non-empty indentation prefix and must be consistent throughout. **Blank lines** — lines of only spaces/tabs — are exempt: their whitespace indents nothing, so it neither sets nor violates the indentation character. Blank lines are ignored outside blocks and retained inside them when they fall between two block lines.

### Guide
`doc/guide.md` is the user-facing tutorial. Every example in it is checked by `guide_test.go` — change both together.

### RFC
The canonical specification is in `rfc/ogdl-rfc.md`; `rfc/analysis.md` records why it says what it says. Consult the RFC for authoritative rules on quoting, escaping, block continuations, and canonical serialization. Appendix A of the RFC is mirrored by `TestVectors` and `TestVectorsIllFormed` in `parser_test.go` — keep the two in step. Appendix B (comment attachment) is mirrored by the tests in `document_test.go`.
