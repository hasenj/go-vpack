# Introduction

VPack is a general purpose library for binary serialization/deserialization
(which we shall refer to as packing/unpacking).

It implements a robust scheme that allows you to be confident you're not making
subtle mistakes which could otherwise cause data corruption.

While this package was originally designed to serve as a building block to
[VBolt][vbolt], it is in fact general purpose and has no dependency on VBolt.

[vbolt]: https://pkg.go.dev/go.hasen.dev/vbolt

## Credits

The techniques described below were largely inspired by the following blog post:

[How Media Molecule Does Serialization](https://handmade.network/p/29/swedish-cubes-for-unity/blog/p/2723-how_media_molecule_does_serialization)

Which itself was informed by the author ([Oswald Hurlem](https://github.com/OswaldHurlem)) asking [Alex Evans](https://x.com/mmalex/) from
[Media Molecule](https://www.mediamolecule.com/) about how to do robust
versioned serialization; the answers from Alex were recorded here:

[gist: mmalex serialization formats](https://gist.github.com/OswaldHurlem/4810ad510669097db872c6de305c9df0)

# High level usage pattern

Given a type T, we define a packing function like this:

```go
func PackT(obj *T, buf *vpack.Buffer)
```

The buffer has a `Writing` flag that determines whether we're reading from it or
writing to it.

This packing function can then be used to both _pack_ and _unpack_ instances of
T to and from a byte buffer using `vpack.ToBytes` and `vpack.FromBytes` or
`vpack.FromBytesInto`.

```go
// t is T
// data is []byte
data := vpack.ToBytes(&t, PackT)
```

```go
// data is []byte
// t is *T
t := vpack.FromBytes(data, PackT)
```

```go
// data is []byte
var t T // empty object
vpack.FromBytesInto(data, &t, PackT)
```

The difference can be subtle, but `FromBytesInto` allows you to program without
littering your code with nil checks.

If an error was encountered during unpacking, FromBytes will return a `nil`
pointer, while `FromBytesInto` will return `false`.

## Example: Packing a struct

Let's look at a concrete - if contrived - example.

```go
type XYZ struct {
	Unit int
	Energy int
	Chapter string
	President bool
}

func PackXYZ(xyz *XYZ, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&xyz.Unit, buf)
	vpack.Int(&xyz.Energy, buf)
	vpack.String(&xyz.Chapter, buf)
	vpack.Bool(&xyz.President, buf)
}
```

The function packs the struct fields in sequence using packing functions
defined in the `vpack` package. Each of these functions itself follows the same
signature: taking a pointer to the item it packs and a pointer to the buffer.

```go
func Int(n *int, buf *Buffer)
func String(s *string, buf *Buffer)
func Bool(b *bool, buf *Buffer)
```

Now let's consider one line in isolations:

```go
	vpack.Int(&xyz.Unit, buf)
```

Depending on the Writing flag on buf, this function will either dump the value
of `xyz.Unit` to the buffer, or it will read data from the buffer and assign it
to `xyz.Unit`.

This same logic applies to all these lines:

```go
	vpack.Int(&xyz.Unit, buf)
	vpack.Int(&xyz.Energy, buf)
	vpack.String(&xyz.Chapter, buf)
	vpack.Bool(&xyz.President, buf)
```

The aggregate effect is that when the Writing flag is on, the entire struct is
written to the buffer, and when the Writing flag is off, the entire struct is
read from the buffer.

## Grasping the architecture

As a user of this package, you almost never have to worry about checking the
`Writing` flag yourself; the functions for packing the primitive types do that
so that you don't have to.

This architecture solves two problems at once:

- You don't need to duplicate the code for reading and writing.
- You don't need to worry about subtle mistakes with catastrophic effects!

Just pack the fields in some sequence, and you're guaranteed that unpacking will
happen in exactly the same order they were written.

## Versioning Data

The first line specifies the version. It doesn't look like the other packing
functions, so why is it there and what does it do?

```go
vpack.Version(1, buf)
```

Robust packing for long term storage requires supporting schema evolution
through a version flag: before writing out the fields, write a version number.

For version 1, we don't do anything special. We just write the version to allow
the code to evolve in the future.

In the future, when we make changes (add fields, remove fields), we surround the
relevant item packing lines with if blocks against version ranges.

Keeping with our example, let's suppose that we decided to remove the `Energy`
field, and add a `Price` field.

```go
type XYZ struct {
	Unit int
	Price int
	Chapter string
	President bool
}
```

How would the packing function change?

We need to consider three things:

- Fields we used to pack, but that no longer exist:
  - Guard with if version <= last version field existed
  - Read into a local varible instead of a struct field
- New fields we added:
  - Guard with if version >= first version field was added
- Changes in data representation
  - Application specific

In this example, since it's contrived, we'll say that in the new model, `Price`
is `Energy * Unit`.

Here's how the packing function would look like:

```go
func PackXYZ(xyz *XYZ, buf *vpack.Buffer) {
	version := vpack.Version(2, buf)

	vpack.Int(&xyz.Unit, buf)

	if version == 1 {
		// version 1 is the last version that had the Energy field
		var energy int
		vpack.Int(&energy, buf)
		// version migration code:
		// In the new version, price is energy * unit
		xyz.Price = energy * xyz.Unit
	}
	if version >= 2 {
		// version 2 is the first version where Price was added
		vpack.Int(&xyz.Price, buf)
	}

	vpack.String(&xyz.Chapter, buf)
	vpack.Bool(&xyz.President, buf)
}
```

You do need to be careful so that the field order does not change between
versions. We introduce a simple testing technique further down.

# The `Buffer` and the `Writing` flag

The basic building block is a `Buffer` struct that wraps a byte slice, and has a
`Writing` flag. This flag is the key innovation that allows this library to be
what it is.

It allows the packing function to fulfill the role of both reading and writing
(serialization and deserialization, or packing and unpacking) at the same time.

If the buffer has the `Writing` flag set, the packing function _writes_ the
binary representation of the object to the buffer. If the flag is not set, it
_reads_ from the buffer into the object passed.

We saw struct packing code that does not check the flag itself; it just calls
more "primitive" packing functions.

These functions do work with the flag. For example, here's the function for
packing a `uint64` using `varint` encoding:

```go
func VInt64(func VInt64(n *int64, buf *Buffer) {
	if buf.Writing {
		buf.Data = binary.AppendVarint(buf.Data, *n)
	} else {
		var err error
		*n, err = binary.ReadVarint(buf)
		if err != nil {
			buf.Error = true
		}
	}
}
```

VPack provides enough primitive packing functions that you generally don't need
to implement your own, but we don't discourage you from implementing your own
primitive packing functions.

You just have to be really careful with them, and
test them very well. Small subtle mistakes here _can_ cause data corruption if
you're not careful!

The most important thing to worry about is the reading code has to consume the
exact same number of bytes that were written by the writing code.

If this condition is not upheld, data _will_ be corrupted.

# Composite objects

VPack provides helper functions to pack slices and maps:

```go
func Slice[T any](list *[]T, fn PackFn[T], buf *Buffer)
func Map[K comparable, T any](m *map[K]T, keyFn PackFn[K], valFn PackFn[T], buf *Buffer)
```

To pack an int slice field `[]int` named `Ids`, you can call:

```go
vpack.Slice(&item.Ids, vpack.Int, buf)
```

You can also use this to packs slices of objects. Suppose you have these types:

```go
type ArticleAuthor struct {
	AuthorId int
	Nickname string
	Contribution string
}

type Article struct {
	ArticleId int
	Title string
	Authors []ArticleAuthor
}
```

Notice the field Authors on Article is a slice of objects.

Here are the packing functions:

```go

func PackArticleAuthor(self *ArticleAuthor, buf *vpack.Buffer) {
	vpack.Int(&self.AuthorId, buf)
	vpack.String(&self.Nickname, buf)
	vpack.String(&self.Contribution, buf)
}

func PackArticle(self *Article, buf *vpack.Buffer) {
	vpack.Int(&self.ArticleId, buf)
	vpack.String(&self.Title, buf)
	vpack.Slice(&self.Authors, PackArticleAuthor, buf)  // <<<<<<
}
```

Notice how we pack the list of Authors using `vpack.Slice`.

## Testing Versioning Code

Earlier we explained versioning on a contrived example, and we mentioned that we
do need to take some care with how code is changed, and that this is one area
where a small mistake could cause problems.

So how do we test the versioning code? Here's a simple idea:

We can duplicate (and rename) our old versions of the struct and serialization
functions

```go
type XYZ_V1 struct {
	Unit int
	Energy int
	Chapter string
	President bool
}

func PackXYZ_V1(xyz *XYZ, buf *vpack.Buffer) {
	vpack.Version(1, buf)

	vpack.Int(&xyz.Unit, buf)
	vpack.Int(&xyz.Energy, buf)
	vpack.String(&xyz.Chapter, buf)
	vpack.Bool(&xyz.President, buf)
}
```

In the test code, we write to a buffer using `PackXYZ_V1`, then we read from the
same buffer using `PackXYZ`, and we check the fields we read to make sure the
data matches what we expect

```go
// helper function inside yor unit test
compareXYZ_V1_V2 := (v1 XYZ_V1, expected XYZ) {
	var packed = vpack.ToBytes(&v1, PackXYZ_V1)

	var actual XYZ
	vpack.FromBytes(packed, &actual, PackXYZ)

	if actual != expected {
		// report error
	}
}
```
