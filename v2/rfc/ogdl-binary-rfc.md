# RFC: OGDL Binary

## Status of This Memo

This document specifies OGDL Binary, a byte-oriented encoding of the Ordered Graph Data Language (OGDL) data model, as an informational specification in the style of an IETF RFC.

It revises the OGDL Binary specification v1.0 published at ogdl.org, whose grammar is reproduced in the source of the reference implementation, `github.com/rveen/ogdl` (`binary.go`). That grammar left open the integer encoding of lengths, the relation between successive levels, text that contains a zero octet, and what a decoder does with a stream that ends early. Section 12 lists where this document differs from the reference implementation, each difference verified against it.

This document depends on `ogdl-rfc.md` ("the core specification"), which defines the OGDL data model.

Distribution of this memo is unlimited.

The key words MUST, MUST NOT, REQUIRED, SHALL, SHALL NOT, SHOULD, SHOULD NOT, RECOMMENDED, MAY, and OPTIONAL in this document are to be interpreted as described in RFC 2119.



## Abstract

OGDL Binary encodes an OGDL tree as a sequence of octets: a header, one record per node in document order, each carrying the node's depth and its content, and a terminating zero. A node's content is either text or a sequence of arbitrary octets. The encoding is self-delimiting, so streams can be concatenated, appended to a file one after another, and read back one at a time. It is the storage format of OGDL logs and the message format of OGDL remote functions.



## 1. Introduction

The text form of OGDL (core specification) is meant to be read and written by people. Programs that store or exchange trees in bulk need a form that is cheap to decode, carries arbitrary octets, and marks its own end. OGDL Binary is that form. It carries the same data model: an ordered tree of nodes, each with a name and an ordered list of children.

```
a          01 47 00            header
  b        01 61 00            level 1, text "a"
  c        02 62 00            level 2, text "b"
d          02 63 00            level 2, text "c"
           01 64 00            level 1, text "d"
           00                  end
```

### 1.1 Relationship to the Earlier Specification

Every stream written by the reference implementation is a well-formed stream under this document and decodes to the same tree, with one exception: a reference stream that ends without its terminating zero is rejected here (section 10), where the reference returns the part it read.

The reference implementation does not write nodes whose name is empty: it omits such a node and writes its children in its place, one level up. This document encodes them (section 5.1). The reference decoder misreads such a node, so a stream that contains one needs a decoder conforming to this document.



## 2. Data Model

A stream encodes the children of an unnamed root, as a text stream does (core specification, section 2): the root itself is not encoded. Each node has a **name**, which is either

- **text**: a sequence of octets other than 0x00, or
- **data**: a sequence of arbitrary octets, possibly empty.

The core data model has only text names. Data names exist so that octets with no text form (images, encrypted values, nested binary streams) can be stored without an encoding layer. An implementation that maps names to typed values (section 2.1) MUST keep the distinction: a node decoded from a data record is data, not text, even if its octets are valid text.

### 2.1 Mapping to Values (Informative)

The Go implementation in this repository decodes a text record to a `Node` whose `Data` is a `string`, and a data record to one whose `Data` is a `[]byte`. It encodes a `[]byte` as a data record, and any other value as a text record holding its textual form (the same form text serialization uses: `42` for the integer 42). Typed values other than strings and octet slices therefore come back as strings, as they do from a text stream.



## 3. Character Model

OGDL Binary is a sequence of octets. Text names SHOULD be UTF-8, as in the core specification, but a decoder MUST NOT reject a stream because a text name is not UTF-8 or contains control characters: the binary form stores what it is given. Such a name may then have no text-stream serialization (core specification, section 4.2).



## 4. Integers

Levels and lengths are non-negative integers encoded in one to four octets. The high bits of the first octet give the length of the encoding; the remaining bits, most significant first, give the value:

| First octet | Octets | Value bits | Range |
|---|---|---|---|
| `0xxxxxxx` | 1 | 7 | 0 – 0x7F |
| `10xxxxxx` | 2 | 14 | 0 – 0x3FFF |
| `110xxxxx` | 3 | 21 | 0 – 0x1FFFFF |
| `1110xxxx` | 4 | 28 | 0 – 0xFFFFFFF |

A first octet of 0xF0 or above is ill-formed (`ErrBinaryInteger`). An encoder MUST use the shortest encoding of a value. A decoder MUST accept a longer encoding of the same value.



## 5. Grammar

```
stream     = header *record end
header     = %x01 "G" %x00
record     = level ( text-node / data-node )
level      = integer                          ; 1 or more
text-node  = [ first-octet *text-octet ] %x00
first-octet = %x02-FF
text-octet  = %x01-FF
data-node  = %x01 *chunk %x00
chunk      = length 1*octet                   ; exactly `length` octets
length     = integer                          ; 1 or more
end        = %x00
```

`integer` is the encoding of section 4. A level of zero is the `end` production: it is the single octet 0x00.

### 5.1 Text Records

A text record is the name's octets followed by 0x00. A text name MUST NOT contain 0x00, and MUST NOT begin with 0x01, which would make the record a data record. An encoder given such a name MUST report an error (`ErrBinaryText`) and write nothing; it MUST NOT silently encode the name as data, which would change it on decoding.

An empty text name is a record consisting of its level and 0x00.

### 5.2 Data Records

A data record is 0x01, then the data in chunks, each preceded by its length, then a zero length (0x00). Chunk lengths are at most 0xFFFFFFF; longer data is split into several chunks. An encoder SHOULD write data in as few chunks as possible. Empty data is 0x01 0x00. The decoded name is the concatenation of the chunks.

### 5.3 Levels

The level of a record is the depth of its node: 1 for a child of the root, one more than its parent's level otherwise. Records appear in document order (pre-order). Therefore the first record has level 1, and each further record has a level between 1 and one more than the previous record's level. A level outside that range is ill-formed (`ErrBinaryLevel`).



## 6. Decoding (Normative)

A decoder reads the header, then records until it reads a level of zero, then stops. It MUST NOT read beyond the terminating zero: the octets after it belong to whatever follows the stream (section 8).

For each record of level `L`, the decoder creates a node with the record's name and appends it to the children of the most recent node of level `L - 1`, or of the root if `L` is 1.

A decoder MUST impose a limit on the level, as the core specification requires of nesting depth (section 9.2 there); a level beyond it is an error (`ErrTooDeep`). The Go implementation uses the same default as its text parser.



## 7. Encoding (Normative)

An encoder writes the header, then a record for each node of the tree in pre-order, then 0x00. A node's level is its depth below the encoded root. It writes no record for the root.



## 8. Concatenated Streams and Logs

Because a stream marks its own end, streams can follow one another in a file or on a connection. A reader of such a sequence decodes one stream, then the next from the octet after the terminating zero, until the input ends.

The input ending exactly where a stream would begin is the normal end of a sequence and is not an error. Any other end of input inside a stream is an error (section 10).

An **OGDL log** is a file of concatenated streams, written only by appending. The position of a stream is the offset of its first octet; given a position, a reader decodes the stream there and learns the position of the next one. Positions are stable, so they can be stored in an index.



## 9. Remote Functions (Informative)

OGDL remote functions exchange trees over a byte stream connection: a request tree, answered by a response tree. Two framings are in use:

- **Protocol 1**: each message is an OGDL Binary stream, delimited by its own end.
- **Protocol 2**: each message is a four-octet unsigned length, most significant octet first, followed by an OGDL Binary stream of exactly that length.

The first child of a request tree names the function. A server that knows no function of that name answers `error notFound`.



## 10. Error Handling

A decoder MUST reject, and report, the following; it MUST NOT return a partial tree:

- a missing or wrong header (`ErrBinaryHeader`);
- an ill-formed integer (`ErrBinaryInteger`, section 4);
- a level out of range (`ErrBinaryLevel`, section 5.3);
- a level beyond its limit (`ErrTooDeep`);
- input ending inside a stream, including before the terminating zero (an unexpected end of input).

Errors SHOULD carry the offset of the octet at which the problem was found.

An encoder MUST reject a text name it cannot encode (`ErrBinaryText`, section 5.1) and a tree deeper than 0xFFFFFFF levels.



## 11. Security Considerations

A data record's length is read before its data. A decoder MUST NOT allocate memory for a length it has not yet read data for: lengths up to 0xFFFFFFF in a short or hostile stream would otherwise cost up to 256 MiB per chunk. The Go implementation grows its buffer as data arrives.

Deep nesting costs memory and, in recursive consumers, stack: decoders limit the level (section 6).

A protocol 2 frame announces its length in advance. A receiver SHOULD refuse lengths above a configured maximum before reading the body.



## 12. Divergences from the Reference Implementation

Each was verified against `github.com/rveen/ogdl` v1.5.

1. **Empty names.** The reference encoder omits a node with an empty name and writes its children one level up; the reference decoder reads an empty text record as text beginning with 0x00 that runs into the next record. Here an empty name is encoded and decoded as such (section 5.1).
2. **Truncated streams.** The reference decoder returns what it read when the input ends early. Here that is an error (section 10).
3. **Level jumps.** The reference decoder silently drops a record whose level is more than one deeper than the previous one. Here that is an error (section 5.3).
4. **Unencodable text.** The reference encoder writes a text name containing 0x00, or beginning with 0x01, producing a stream that decodes differently. Here that is an error (section 5.1).
5. **Remote functions, protocol 2.** The reference reads a frame's header and body with single reads, which fail when the connection delivers a message in parts. That is an implementation defect rather than a protocol difference; the Go implementation here reads until the frame is complete.



## 13. Conformance

A conforming decoder decodes every well-formed stream of Appendix A to the tree shown, and rejects every stream of Appendix A.1. A conforming encoder produces, for each tree of Appendix A, exactly the octets shown.



## Appendix A. Test Vectors

Octets in hexadecimal. Trees in canonical text form; `<data: …>` denotes a data name.

| Tree | Stream |
|---|---|
| (empty) | `01 47 00 00` |
| `a` | `01 47 00 01 61 00 00` |
| `a` with children `b`, `c`; then `d` | `01 47 00 01 61 00 02 62 00 02 63 00 01 64 00 00` |
| `x` with child `<data: 00 01 02>` | `01 47 00 01 78 00 02 01 03 00 01 02 00 00` |
| `n` with child `42` | `01 47 00 01 6e 00 02 34 32 00 00` |
| `""` with child `child` | `01 47 00 01 00 02 63 68 69 6c 64 00 00` |
| `<data: empty>` | `01 47 00 01 01 00 00` |
| a node at depth 128 (127 ancestors `x`), named `y` | … `7f 78 00 80 80 79 00 00` |
| `<data: 200 × 7a>` | `01 47 00 01 01 80 c8` then 200 × `7a`, then `00 00` |

The first five rows, the depth-128 row and the 200-octet row are what the reference implementation writes for the same trees. The `""` and empty-data rows are not: the reference writes `01 47 00 01 63 68 69 6c 64 00 00` (the child moved up) for the first and cannot represent the second.

### A.1 Ill-Formed Streams

| Stream | Error |
|---|---|
| (empty input) | end of input (the normal end of a sequence, section 8) |
| `01 47` | unexpected end of input |
| `01 48 00 00` | `ErrBinaryHeader` |
| `01 47 00 01 61 00` | unexpected end of input (no terminating zero) |
| `01 47 00 01 61` | unexpected end of input (inside a record) |
| `01 47 00 02 61 00 00` | `ErrBinaryLevel` (first level is 2) |
| `01 47 00 01 61 00 03 62 00 00` | `ErrBinaryLevel` (1 to 3) |
| `01 47 00 f0 61 00 00` | `ErrBinaryInteger` |
| `01 47 00 01 01 05 61 62 00` | unexpected end of input (chunk shorter than its length) |
