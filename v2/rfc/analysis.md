# Analysis of rfc/ogdl-rfc.md

> **Status: applied.** Every point below has been incorporated into
> `ogdl-rfc.md`. This document is retained as the rationale record for those
> changes; section numbers in Parts 1 and 2 refer to the *pre-revision* draft.
> Part 4 records a later revision that replaced the chain rule of §1.1 with
> head-and-arguments lines and pairs; read §1.1 and §3.3 as history. Part 5
> withdraws the pairs again; read the pair material of Part 4 as history too.

Review of the OGDL RFC draft: defects and gaps. The inline (parenthesis/comma)
syntax has been ruled out of scope; Part 3 records that decision and its
consequences. Section numbers refer to `ogdl-rfc.md`.

---

## Part 1 — Blocking defects

These are places where the document is self-contradictory or where two conforming
implementations would reasonably produce different trees for the same input.

### 1.1 The core structural rule is never stated (§2.1, §5)

The single most important rule in OGDL — *the elements of a line form a descending
chain, and the next line's indentation attaches to the last of them* — appears
nowhere normatively. §2.1 says only "each line defines one or more *nested* nodes".
§5 gives `list = element (space element)*` with no semantics attached.

So `a b c` producing `a → b → c` is folklore. Worse, the follow-on question is
entirely undefined: given

```
a b c
  d
```

does `d` attach to `c` (last element), to `a` (the line's indentation owner), or is
it an error? The Go implementation answers "`c`", via the `indent*64` pseudo-depth
hack in `parser.go:71` — an undocumented invention that also silently overflows
into nonsense for long lines.

**Required:** a normative subsection defining *virtual depth*. Something like: the
k-th element of a line at indentation depth n has virtual depth `(n, k)` ordered
lexicographically; the existing stack rules in §6.5 then operate on virtual depths
unchanged. This also gives you the exact hook the inline syntax needs (Part 3).

### 1.2 Multi-line quoted strings are used but never defined (§4.5 vs §7)

§7 states "Multi-line strings are double quoted (block is defined as
non-canonical)" — i.e. canonical serialization *depends* on multi-line quoted
strings. But §4.5 defines `quoted_char = char_quoted | escape` where
`char_quoted = char - char_end - "'" - '"'`, which **includes** LF and CR only
by accident (they are not excluded), and no rule says what a break inside a quoted
string means, whether CRLF is normalized, or what happens to the indentation of
continuation lines.

The implementation (`lexer.go:171-190`) normalizes any break to LF and then strips
*all* leading whitespace on the continuation line — meaning a multi-line string can
never contain a line that begins with a space, and the amount stripped does not
relate to the enclosing indentation. None of that is in the spec, and it is not
obviously the right rule.

There is also an internal contradiction: §7 says "one node per line", yet canonical
output for a multi-line name spans several lines.

**Recommended fix:** add `\n` and `\t` to the escape set in §4.5, and make canonical
serialization single-line (escape breaks rather than emit them literally). That
restores "one node per line" as a true invariant, makes canonical OGDL trivially
line-splittable, and removes the whole continuation-indentation question. Keep
literal multi-line quoting as a *non-canonical* input convenience, with the
stripping rule written down explicitly.

### 1.3 Indentation-character determination is contradictory (§6.3 vs §6.6)

§6.3: the indentation character is fixed by "the first non-empty, non-comment line
that contains indentation" — comment lines are excluded from *determining* it.
§6.6: comment lines "MUST still respect the indentation character consistency rule".

So a TAB-indented comment line preceding all SPACE-indented content is
simultaneously exempt (can't set the character) and bound (must match a character
not yet determined). Undefined.

**Fix:** determine the character from the first line with a non-empty indentation
prefix, full stop, including comment and otherwise-empty lines. Simpler, and
implementable in one pass with no lookahead.

### 1.4 Dedent to a non-existent level is left open (§6.5, §9)

§9 literally contains `(??)`. The case is:

```
a
    b
  c
```

`c` dedents to a depth that was never on the stack. §6.5 rule 4 ("well-nested")
implies error; the implementation accepts it and makes `c` a sibling of `b`
(verified). Pick one and make it normative. Recommendation: **error**, matching
Python and matching the "well-nested stack" language already present. Lenient
recovery can be a MAY.

### 1.5 The EBNF in §5 cannot parse a document

Three separate problems:

- `line(n)` has no terminating break and `tree = line* char_end` has no separator,
  so the grammar as written accepts only a single line.
- `block(n) = space "\\" ...` requires a *preceding* space, so a block cannot begin
  a line, and `block(n)?` only attaches after a non-empty `list`. §8 ("a block
  begins with a backslash") contradicts this.
- `comment` in §4.6 is defined as `(space | break) "#" ...` — it swallows its own
  leading delimiter, but §5 uses it as `line(n) = space(n) (... | comment)`, which
  double-counts the space. A comment as the first thing in a stream (no preceding
  space or break) does not match at all.

The grammar needs one pass to make it composable, with the delimiters factored out
of `comment` and a `break` in `line`.

### 1.6 Comment rule vs implementation (§4.6)

Spec: `#` is a comment when preceded by space *or break* and followed by a space.
Implementation only recognizes comments at line start (`lexer.go:129`), so
`a b # comment` parses `#` and `comment` as ordinary word nodes (verified). One of
the two must move. Note the spec's version is the more useful one (trailing
comments) but requires the lexer to track "previous token was space".

Also: `# ` followed by nothing (`#` + space + EOL) fails `string = char_string+`, so
an empty comment is not a comment. Almost certainly unintended.

### 1.7 Mixed indentation is specified but not enforced

§6.3/§6.4 are emphatic (three MUSTs, an explicit error list) and `ErrMixedIndentation`
exists in `parser.go:32` — but is never returned. `"a\n\tb\n  c\n"` parses happily
(verified). Either the spec relaxes or the parser gets the check; the `// TODO Check
TAB/SP coherency` at `parser.go:64` says which was intended.

---

## Part 2 — Gaps and smaller improvements

**Data model**

- **The root is never mentioned.** A document with several top-level nodes is a
  *forest*; every implementation invents a synthetic root (`_` here). Say so
  normatively — including whether the root's name is significant (it isn't) and
  whether it can be serialized.
- **Duplicate sibling names are legal.** One sentence, because every reader arriving
  from JSON/YAML will assume otherwise.
- **The empty name.** `""` is producible by the serializer and round-trips, but §4.3
  `word` requires ≥1 character and nothing says a node may have an empty name. State
  it.
- **Comments carry no data.** Say explicitly that comments are discarded and are not
  part of the tree — otherwise round-trip/canonicalization obligations are unclear.
- **Document vs stream.** Is a stream one document? Is there a document separator?
  §11 registers a media type, so this needs an answer.

**Character model**

- **U+FEFF (BOM) is unhandled.** As written, a leading BOM becomes part of the first
  word. Add: a leading U+FEFF MUST be ignored (or MUST be rejected — but decide).
- `char_end` mixes a character class with "end of input". Split out an `EOS` terminal;
  the current definition makes `char_word = char - char_break - char_end - char_space`
  subtract a non-character.
- §3.1 says "except control characters (Cc)" but §3.2 defines `char_control` as
  `U+0000..U+001F | U+007F..U+009F`, which happens to equal Cc — say that it is Cc
  and give the ranges as informative.
- Unpaired surrogates / invalid encoding: §10's "validate Unicode input" should be a
  concrete MUST-reject rule.

**Lexical**

- §4.3 `word = (char_word - "'" - '"') char_word*` means quotes are legal *inside* a
  word (`don't`, `a"b`) but not at the start. That is a deliberate and slightly
  surprising rule; add an example, and state that a word is **not** terminated by a
  quote (so `a"b c"` is two words `a"b` and `c"`).
- §4.4 `string` is defined but used only inside blocks; say so or inline it.
- §4.5 "Any other sequence starting with `\` is interpreted literally (the backslash
  is preserved)" — this permanently forecloses adding `\n`/`\uXXXX` later without a
  breaking change. If you want the escape set to be extensible, make unknown escapes
  an **error** (or reserved) now. This matters more than it looks.

**Canonical form (§7)**

Underspecified for something that carries interoperability weight. Add: breaks are
LF; the stream ends with exactly one LF; no trailing whitespace on any line; no
comments; no blocks; no BOM; and — most importantly — a statement that
canonicalization is a *function* on the tree, so two documents are equivalent iff
their canonical forms are byte-identical. That sentence is what makes a conformance
test suite possible.

**Error handling (§9)**

SHOULD is too weak for well-formedness. Well-formedness violations MUST be rejected
by a conforming parser; a separate "lenient/recovering mode" MAY be offered but MUST
NOT be the default. Also add: nesting-depth and length limits are implementation-
defined but MUST be documented (ties §10 to conformance).

**IANA (§11)**

- `text/*` with an optional `charset` and no stated default is an interop trap.
  Recommend: charset defaults to UTF-8; UTF-8 MUST be used for interchange.
- Fragment identifiers: "not defined by this specification" is fine, but consider
  *reserving* them instead, so a future addressing/path scheme has somewhere to go.
- Missing: author, change controller, and a stable reference — required for actual
  registration.

**Missing sections**

- **Conformance**: what a "conforming parser"/"conforming producer" must do, and the
  profile levels (see Part 3).
- **Versioning/extension policy**: how a future version signals itself, which
  characters are reserved for future use. This is the hinge for the inline question.
- **Test vectors**: even a short appendix of input → canonical output pairs would
  pin down §1.1–§1.4 above better than prose.

---

## Part 3 — Inline syntax: out of scope

**Decision: the inline (parenthesis/comma) syntax is dropped from this specification
and is not being specified elsewhere at this time.** This section records the
consequences so the decision is not silently re-opened later.

### 3.1 What this means lexically

`(`, `)` and `,` remain ordinary members of `char_word`. They are content, with no
structural meaning, in every position:

```
gw 192.168.0.1, 192.168.0.2      # words "192.168.0.1," and "192.168.0.2"
note (see below)                 # words "(see" and "below)"
handler f(x)                     # one word "f(x)"
```

This is the status quo and matches the current implementation; nothing in the RFC
needs to change to hold this position. The spec SHOULD say it explicitly, though —
one sentence in §3 or §4 stating that no character other than SPACE, TAB, LF, CR,
`#`, `"`, `'` and `\` has lexical significance. Right now that is only inferable
from the character-class algebra, and readers arriving from ogdl.org will assume
otherwise.

### 3.2 The compatibility consequence

Introducing inline syntax later removes `(`, `)`, `,` from `char_word`. That is a
breaking change to any document containing them unquoted, and it cannot be made
additive: a parser either lexes `(` structurally or it does not, so two
implementations disagreeing while sharing the media type `text/ogdl` is a hard
interop failure rather than graceful degradation.

Therefore, if inline syntax is ever adopted, it will require a version signal.
That is the practical reason to add the **versioning/extension policy** section
already listed in Part 2 — it is what makes a future reversal possible at a defined
cost instead of an undefined one. Reserving the characters pre-emptively was the
alternative; that option is now closed by choice, which is fine, but the escape
hatch should exist on paper.

### 3.3 What remains worth doing anyway

One recommendation from the inline analysis stands on its own merits and should be
kept: **defining tree construction over *virtual depth* (Part 1, §1.1)** rather than
over raw line indentation. It is required regardless — it is what makes `a b c` and
its indented children well-defined today. That it would also have made inline syntax
a two-page addendum is now merely incidental.

Also independent of this decision: the RFC should state the closed set of lexically
significant characters (§3.1 above). That single sentence settles `.`, `[`, `]`, `{`,
`}`, `(`, `)` and `,` at once — any addressing or expression notation built on them is
an application-level interpretation of a node *name*, and must not leak into OGDL's
lexis.

---

## Part 4 — Line structure revised: head and arguments, and pairs

**Decision: a line is a head followed by its arguments, and `key=value` is a
pair.** This supersedes the chain rule of §1.1 and the virtual-depth machinery
built for it. It is recorded here because it reverses an earlier, deliberate
decision.

### 4.1 Why the chain rule went

The chain rule (`a b c` is `a → b → c`) was consistent, and §1.1 made it
precise. Precision did not stop it surprising people. `S1 value 1.0 class asil`
looks like a record and is a chain four deep, and that mismatch caused a real
bug (`ogdl-note.md`) despite the rule being documented. The rule optimized for
path notation inside a document — writing `a b c` to mean the path `a.b.c` —
which nobody does now that paths have a syntax of their own (`ogdl-path-rfc.md`).
Canonical form never used inline chains at all, so the chain rule served only
hand-written documents, and those were exactly the ones it misled.

### 4.2 Alternatives considered

- **Parentheses and commas** (`S1(value 1.0, class asil)`), the original inline
  syntax. Unambiguous, but it adds two delimiters per node and one per field,
  and makes `(`, `)` and `,` significant inside words (`f(x)`, `1,2`), against
  OGDL's main aim: few visible separators. Ruled out again for the reasons of
  Part 3.
- **Keep the chain, document it harder.** Already tried.
- **Siblings only, no pairs.** Fixes `rule allow any`, but leaves no way to
  write a record on one line; `S1 value 1.0 class asil` becomes four flat
  siblings, which is less wrong than a chain but still not what was meant. No
  layout can tell a field name from a positional value, so *some* marker is
  needed, and one character per field is the least that works.
- **Pairs with `:`.** `:` occurs in times, URLs and IPv6 addresses, all common
  values. `=` is much rarer in data.

### 4.3 Why the key class is narrow

`=` is significant only after a key of letters, digits, `_` and `-` (not first),
and only when a value follows that does not begin with `=`. That keeps the most
common `=`-bearing words as plain words: comparisons in rule predicates
(`S>=S2`, `a==b`), command-line flags (`--flag=x`), query strings and base64
padding (`abc=`). A wider class would have bought keys with punctuation, which
can always be written on their own line, at the price of quoting many values.
`.` is excluded so that `a.b=1` cannot be mistaken for a path assignment.

### 4.4 What changed, and what did not

- Canonical form is unchanged — it never had inline elements — so every
  canonical document means what it meant before. Only hand-written lines with
  more than one element, or two elements followed by an indented line, change
  meaning.
- Tree construction is simpler: a stack of `(node, depth)`, with only heads
  pushed. Virtual depth is gone.
- Minimal quoting gains one case: a name that would be read as a pair.
- A compact form (RFC §8.5) is now possible, because a node's leaf children and
  single-valued children can be written on its line unambiguously. It is
  presentation, not equality.
- Path, expression and template layers operate on trees and are unaffected.

### 4.5 Comments

§2.5's "a conforming parser MUST discard" comments was stronger than needed.
What matters is that comments are not data: they create no nodes and do not
affect equivalence. Whether a parser keeps them beside the tree is a feature of
the parser, and keeping them is what lets a program rewrite a hand-maintained
file. The RFC now permits it and gives an attachment rule (Appendix B) so that
implementations agree on where a comment goes.

---

## Part 5 — Pairs withdrawn

**Decision: `=` is ordinary content again. A line is a head and its leaf
arguments, nothing more.** This reverses the pair half of Part 4; the
head-and-arguments half stands.

### 5.1 Why pairs went

- **One element, two nodes.** Everywhere else in OGDL an element is one node,
  and elements are separated by whitespace. A pair was the only element that
  denoted two nodes, with no space between them. The RFC had to say so in bold.
- **A rule for one level only.** Without pairs, the structure rule is the same
  at every depth: a line holds one node and its leaf children, and each
  further level is a further line. Pairs added a third level to a line, but
  only for a leaf value, and it composed with nothing: `a=b=c` did not nest,
  a pair's value could not have children, and nothing indented could attach
  beneath it. Blocks remain the one exception to "one level per line", and
  they are leaves by nature.
- **Recognition depended on context.** Whether `=` split a word depended on the
  key class (§4.3 above): `flag=x` was a pair but `--flag=x` a word, `x_y=1`
  a pair but `x.y=1` a word, `k=\` a pair whose value was `\` though a bare
  `\` opens a block. Each case had a reason, but a reader could not guess
  them. That is the failure that removed the chain rule (§4.1): precise, and
  still surprising.
- **Cost out of proportion.** A token type, lexer state carried across tokens,
  a minimal-quoting case, a compact-form case with its own extra quoting rule,
  four test vectors, and a schema rule (Schema RFC 3.4) existing only to catch
  `port?=!int`.

### 5.2 What is lost

A record no longer fits on one line. `S1 value=1.0 class=asil` becomes

```
S1
  value 1.0
  class asil
```

which is one line per field — the layout canonical and compact form already
used for such a node, so the written form and the serialized form now agree.
Part 4.2 argued that "some marker is needed" because layout cannot tell a field
name from a positional value. That remains true of a single line; the answer
is that a line is the marker. `S1 value 1.0 class asil` reads as four flat
siblings, which errs towards too little structure and never invents any.

### 5.3 What changed

- `=` leaves the list of significant characters (RFC §3.3), which shrinks to
  space, tab, breaks, quotes, `\` and `#`.
- Canonical form is unchanged except that names containing `=` are no longer
  quoted. A canonical document produced under pairs still parses to the same
  tree; `"a=b"` is merely over-quoted.
- Compact form inlines leaf children only.
- Hand-written documents that used pairs change meaning: `k=v` is now one
  node named `k=v`. The format is still unregistered (RFC §10), which is why
  the change is made in place.
- The schema language keeps rejecting an unescaped `=` in a particle name: a
  schema written under pairs, `port=!int`, would otherwise silently describe
  a node named `port=!int`.

---

## Priority ordering

1. §1.1 virtual-depth rule (everything else depends on it)
2. §1.2 multi-line strings — canonical form currently rests on undefined behavior
3. §1.3, §1.4, §1.6 — pick the normative answer, fix parser or spec to match
4. §1.5 rewrite the EBNF so it composes
5. §1.7 enforce mixed indentation, or relax the spec
6. Add Conformance + versioning/extension policy + test-vector appendix
7. §3.1 state the closed set of lexically significant characters
