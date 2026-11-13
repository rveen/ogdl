# Moving from ogdl v1 to v2

`github.com/rveen/ogdl` (v1) and `github.com/rveen/ogdl/v2` are separate
modules and can be used side by side, so a program can move one package at a
time. v2 is a rewrite against a set of specifications (`rfc/`); where it
behaves differently from v1, the RFCs say so, and most differences are v1
behaviour that was never specified.

What stays the same:

- **The text format.** A v1 document is a v2 document. v2 rejects malformed
  input that v1 accepted silently (mixed indentation, invalid UTF-8, control
  characters): see `rfc/ogdl-rfc.md`.
- **The binary format.** Whatever v1 writes, v2 reads to the same tree and
  writes back byte for byte (`rfc/ogdl-binary-rfc.md`, checked on 76,770
  stored objects). v2 also encodes nodes with an empty name, which v1 cannot
  read.
- **Remote functions.** v1 and v2 clients and servers talk to each other, in
  both framings.

## The tree

| v1 | v2 | Notes |
|---|---|---|
| `*ogdl.Graph` | `*ogdl.Node` | `This` is now `Data`; `Out` is unchanged. |
| `ogdl.New(nil)`, `ogdl.New(x)` | `&ogdl.Node{}`, `&ogdl.Node{Data: x}` | |
| `g.Add(x)` (returns the child) | `c := &ogdl.Node{Data: x}; n.Out = append(n.Out, c)` | No method yet. |
| `g.Node(name)` | `n.S(name)` | Both look at **direct children only**; `Node` is not a path. `Node` returns nil when nothing matches, `S` an empty node. |
| `g.Get(path)` | `n.Get(path)` | Same node for the same path. v1 returns nil when nothing matches, v2 an empty node (never nil); `GetErr` reports a malformed path. |
| `g.Len()` | `len(n.Out)` | v1 returns -1 for nil. |
| `String()`, `ThisString()`, `Int64()`, `Float64()`, `Bool()` | the same | Same results. |
| `Text()` | `Text()`, `Canonical()` | v2 ends the text with a line break and quotes canonically (`Canonical` returns the error `Text` drops). |
| `Set(path, v)` | `Set(path, v)` | Same tree for a plain path; `rfc/ogdl-expr-rfc.md` defines assignment. |
| `ogdl.FromString(s)`, `FromBytes(b)` | `ogdl.ParseString(s)`, `ogdl.Parse(bytes.NewReader(b))` | Return an error for malformed input. |
| `ogdl.FromFile(f)` | `os.Open` then `ogdl.Parse(r)` | |
| `g.Eval…`, `ogdl.NewExpression` | `n.EvalString(s)`, `ogdl.ParseExpr(s)` | `rfc/ogdl-expr-rfc.md`, section 12, lists the differences. |

Not in v2 (yet): `Clone`, `Copy`, `Delete`, `AddNodes`, `Create`, `GetAt`,
`Show`, `FromJSON`, and the character-class helpers (`IsLetter` and others).

Paths: plain paths (`a.b.c`, `a[1]`, `a{1}`) select the same nodes as v1's
`Get`. v2 differs deliberately in places on indexing and on `{}`: read
`rfc/path-notes.md` before relying on anything beyond plain paths.

## Binary data and logs

| v1 | v2 |
|---|---|
| `g.Binary()` | `n.Binary()` (nil on error), `n.MarshalBinary()`, `n.WriteBinary(w)` |
| `ogdl.FromBinary(b)` | `ogdl.ParseBinary(b)`: `(*Node, error)`; exactly one stream |
| `ogdl.FromBinaryReader(r)`, `FromBinaryBufioReader(br)` in a loop until nil | `r := ogdl.NewBinaryReader(br)`, then `r.Read()` until `io.EOF` |
| `ogdl.FromBinaryFile(f)` | `os.ReadFile` then `ParseBinary` |
| `ogdl.OpenLog(f)`, `&ogdl.Log{}` | the same |
| `log.Add(g)` returns the position | `log.Add(n)` returns `(position, error)` |
| `log.Get(pos)` returns a nil tree at the end | `log.Get(pos)` returns `io.EOF` at the end |

A truncated or malformed stream is an error in v2 (`*BinaryError`, with the
offset), where v1 returned whatever it had read.

## Remote functions (`ogdlrf`)

The types keep their names. `Client.Timeout` and `Server.Timeout` are
`time.Duration` (v1: seconds as an `int`). A `Function` takes and returns
`*ogdl.Node`. `Server.Close` stops a server; `ServeListener` serves on a
listener you made. A `Client` may be shared between goroutines.

## Templates

| v1 | v2 |
|---|---|
| `ogdl.NewTemplate(s).Process(g)` (`[]byte`) | `t, err := ogdl.ParseTemplate(s)`, then `t.Process(ctx)` (`string, error`), or `ogdl.ProcessString(s, ctx)` |

Template text changes:

1. **Loop variables are the element.** `$for(item, list)` binds `item` to the
   element, so `$item[0].x1` becomes `$item.x1`, `$file[0][0].name` becomes
   `$file[0].name`, and a bare `$f[0]` becomes `$f._name`. The binding ends
   with the loop and is never written into the context.
2. **`$\`** (a literal `$`) becomes **`$$`**.
3. **Errors are reported.** v1 read these silently; v2 rejects the template:
   - an unclosed `$(`: v1 closes it at the end of its line and drops the line
     break;
   - a `$end` with no open `$if` or `$for`: v1 stops rendering there;
   - an unclosed `$if` or `$for`: v1 renders nothing;
   - a path followed by `.` (`is $x.`): v1 drops the `.`. Write `${x}.`.
4. A `$` not followed by a path, `(`, `{` or `$` is literal text in v2. v1
   dropped some of them (`['$', '$']` in a MathJax configuration).

Points 1 to 3 (the unclosed `$(`) can be applied mechanically; the template
RFC (`rfc/ogdl-template-rfc.md`, section 10) gives the exact rule. The rest
need a decision per template.

**HTML escaping** is a mode chosen by the program (`ParseHTMLTemplate`), off
by default, so templates render as before. Before turning it on, every Go
function that returns markup must return the `ogdl.HTML` type.
