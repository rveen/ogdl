# To do

## Binary OGDL

Done: `rfc/ogdl-binary-rfc.md`, `binary.go` (encoding and decoding),
`log.go` (the log-file layer) and `ogdlrf/` (remote functions). Checked
against `github.com/rveen/ogdl` v1.5 on 76,770 objects from KF databases and
`.gb` files: same trees, byte-identical re-encoding; the remote functions
interoperate with v1 in both protocols.

## Built-ins applied to a set

`firewall.rule{}._len` returns the child count of the *last* rule, not a
count per rule nor the number of rules. In `evaluator.evalPath`
(`eval.go`), when a member resolves to a host value for each node of a set,
each result overwrites the previous one in `host`, so the last member wins
silently. The same applies to `_string`, `_thisString` and `_raw`, and to host
members reached through a set.

To do:

- Decide the meaning in the expression RFC (section 8.4): most useful is that
  a built-in after a set describes the set as a whole, so `a{}._len` is the
  number of matches; mapping over the members would need a value for "a list
  of results", which the value model does not have.
- Fix the evaluator to match, or report an error, but never pick the last
  member silently; add tests.

## Go values in templates and expressions

Evaluate how far navigation into arbitrary Go values should go, so that a
template context need not be an OGDL tree. A good part already works through
`reflect.go` (checked 2026-10-07): with Go values stored in the context,
`$tags[1]`, `$for(t, tags)`, `$u.Name`, `$u.Tags[0]`, `$u.Addr.City`,
`$u.Meta.age`, `$u.Greet()`, `$m.a[1]`, `$users[1].Name` and
`$for(x, users)<$x.Name>$end` all render as expected, and a nil pointer
renders as nothing.

Gaps found:

- **Maps and structs are not iterable.** `$for(k, u.Meta)` and `$for(f, u)`
  fail with "value is not iterable". Iterating a map needs a defined order
  (sorted keys?), since Go's is random, and a way to reach both key and value.
- **Built-ins do not see Go values.** `$u.Tags._len` renders nothing, and
  `$tags._len` renders 1 — the length of the node wrapping the slice, not of
  the slice. The expression RFC (section 8.4) applies built-ins to nodes only.
- **Composite values render with Go's `fmt`.** `$tags` gives `[red green]`
  and a struct gives `{Ana [x y] 0xc0000286d0 map[age:30]}`, pointer address
  included. An OGDL-style rendering (canonical or compact) would be more
  useful and stable.
- **Assignment does not write into Go values.** `$(u.Name = 'Bo')` creates a
  child node `Name` under `u` that shadows the field; the struct is untouched.
  Consistent with "the tree shadows the host value", but possibly surprising.

Options to weigh:

1. Extend the reflection layer (iteration, built-ins, rendering) case by case.
2. Convert Go values into a `Node` tree up front (a reflection-based
   `FromValue`), so that everything downstream only knows trees. Simpler
   semantics, at the cost of a copy, of losing methods, and of handling
   cycles.
3. A mix: convert lazily, on first navigation.

Whatever is chosen belongs in the expression and template RFCs first, with
tests for each case above.
