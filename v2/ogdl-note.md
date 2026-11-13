# OGDL design note: inline syntax tension and the case for parentheses

> **Resolved.** The chain rule described below has been replaced. A line is now
> a head followed by its children as siblings, all leaves, and a record takes
> one line per field:
>
> ```ogdl
> S1
>   value 1.0
>   class asil
> ```
>
> Parentheses were not reintroduced. A `key=value` pair was tried for one-line
> records and withdrawn, because it made a line's structure depend on a marker
> inside a word. See RFC §6.5, and `rfc/analysis.md` Parts 4 and 5 for the
> reasoning. The note is kept as the record of the problem.

## OGDL's core strength

OGDL's fundamental design decision — an ordered list of named nodes at every level — is what distinguishes it from hash-table-oriented formats like TOML, JSON, and technically YAML.

In a hash-table format you are forced to choose between two incompatible types: unordered maps (named children) and ordered sequences (lists). This choice is structural rather than semantic, and it leads to awkward encodings whenever you need both naming and ordering at the same level.

OGDL has one type: an ordered list of named nodes. Every level is simultaneously named *and* ordered. This matters concretely in the risk2 configuration for Truke KF:

- **Label ordinals** derive from declaration order. The predicate `S>=S2` compares positions in a sequence; if label names were hash keys the comparison would be undefined.
- **Rule evaluation** is `first_match` over an ordered sequence. Unordered rules would make a `first_match` strategy meaningless.
- **The combined ASIL+AP scale** works because ASIL labels at ordinals 10–12 are guaranteed to follow AP labels at 0–9. That guarantee is OGDL's, not the application's.

TOML explicitly states its goal as mapping to a hash table, and consequently cannot represent an ordered list of named items without encoding the ordering separately — which is exactly the accidental complexity OGDL avoids.

---

## The tension in the inline syntax

OGDL's canonical multi-line form is clean and unambiguous. Given a label with three attributes:

```ogdl
S1
  value 1.0
  class asil
  title "S1 – Slight injury"
```

`value`, `class`, and `title` are three sibling children of `S1`. This is consistent with the ordered-graph model.

The inline form is where the tension appears. The following looks like it should be the compact equivalent of the above:

```ogdl
S1  value 1.0  class asil  title "S1 – Slight injury"
```

But OGDL's inline syntax creates a **chain**, not siblings:

```
S1 → value → 1.0 → class → asil → title → "S1 – Slight injury"
```

Each token becomes a child of the previous. `title` ends up three levels deep under `asil`, not as a sibling of `class`. The inline form and the canonical form are not semantically equivalent for multi-attribute nodes.

This caused a real parsing bug in `risk2/parser.go`: a label written with both `class` and `title` on one line would silently drop the `title`, because the parser looked for `title` at the wrong level in the chain. The fix required a general chain walker that traverses the attribute sequence explicitly.

### Root cause

OGDL's inline syntax serves two use cases with conflicting intuitions:

1. **Path / traversal notation** — `a b c` means "traverse from `a` to `b` to `c`." A chain is exactly right. This is OGDL's original graph-navigation use case.

2. **Configuration shorthand** — `node attr1 val1 attr2 val2` looks like a record with named fields. The natural reading is siblings, not a chain.

The tension is not a bug in the chain model — that model is internally consistent for graph traversal. The problem is that configuration users bring the record-field intuition to a path-navigation syntax. The two collide when more than one attribute appears on a single line.

---

## The removed inline format

OGDL originally included an inline format using parentheses and commas:

```ogdl
S1(value 1.0, class asil, title "S1 – Slight injury")
```

This would have been unambiguous: parentheses delimit the children of `S1`, commas separate siblings. No chain, no depth confusion. The inline and canonical forms would have been genuinely equivalent — just different whitespace conventions for the same graph structure.

The community rejected this syntax, most likely on aesthetic grounds: "it looks like a function call," or "it looks like JSON," or "it breaks OGDL's minimalism."

This is a case where the aesthetic objection overrode a real semantic need. Rejecting the parentheses form did not make OGDL simpler — it made the *appearance* simpler while leaving the semantic ambiguity intact. Users and parser authors must then reconstruct the intended meaning through convention and workarounds.

---

## The case for reintroduction

The parentheses-and-commas form deserves to be reconsidered, for several reasons:

**It solves the real problem.** The chain-vs-siblings ambiguity is not a corner case; it surfaces whenever a configuration author writes a multi-attribute node compactly. The parentheses form eliminates the ambiguity entirely.

**It preserves OGDL's model.** The parentheses are syntactic sugar over the same ordered-named-node structure. The canonical form and the parentheses form would mean identical things; the parser would produce the same graph. Nothing in the language model changes.

**Users who find it ugly can ignore it.** The canonical multi-line form remains valid and is arguably preferable for complex nodes. The parentheses form is strictly additive — an opt-in compact notation for authors who want it.

**It is not JSON.** JSON uses `{}` for unordered maps and `[]` for ordered arrays. OGDL with `()` would use one construct for ordered named children, which is a different and simpler model.

**Practical recommendation:** introduce parentheses as an optional inline syntax, documented explicitly as "siblings," with the chain syntax retained for path notation. Restrict the unparenthesised inline form to single-value nodes (`name "config"`, `1 value 0.1`) and require either canonical multi-line or parenthesised inline for nodes with multiple attributes. This eliminates the ambiguity without breaking anything.

---

## Summary

| Form | Semantics | Ambiguity |
|------|-----------|-----------|
| Canonical multi-line | Siblings ✓ | None |
| Inline chain (current) | Chain (not siblings) | Yes — looks like siblings |
| Inline parentheses (removed) | Siblings ✓ | None |

The removed parentheses syntax was the right design. The community's aesthetic preference for surface simplicity sacrificed semantic clarity. Reintroducing it — as an optional, explicitly documented form — would close the gap between what the inline notation looks like it means and what it actually means.
