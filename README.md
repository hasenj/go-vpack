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

Given some type of struct like:

```go
type XYZ struct {
	Unit int
	Energy int
	Chapter string
	President bool
}
```

Define a function that receives two parameters:

-  Pointer to the object
-  Pointer to the buffer

In the function body, pack the fields serially one after the other.

```go
func PackXYZ(xyz *XYZ, buf *vpack.Buffer) {
	vpack.Int(&xyz.Unit, buf)
	vpack.Int(&xyz.Energy, buf)
	vpack.String(&xyz.Chapter, buf)
	vpack.Bool(&xyz.President, buf)
}
```

This function can be used to both _pack_ the object into a byte buffer, and
_unpack_ it from a byte buffer.

To pack the object into a byte buffer, call `vpack.ToBytes`, passing it a
pointer to the object you want to pack, and the packing function.

```go
data := vpack.ToBytes(&xyz, PackXYZ)
```

To unpack the object from the byte buffer into an object, call `FromBytes`,
passing it the byte slice and the packing function:

```go
xyz := vpack.FromBytes(data, PackXYZ)
```

You can also call `FromBytesInto` to unpack into a local variable.

```go
var xyz XYZ // empty object
vpack.FromBytesInto(data, &xyz, PackXYZ)
```

The difference can be subtle, but FromBytesInto allows you to program without
littering your code with nil checks.

If an error was encountered during unpacking, FromBytes will return a `nil`
pointer, while FromBytesInto will return `false`.

Notice how the functions to pack the primitive fields themselves follows the
same signature, taking a pointer to the primitive type and a pointer to a
Buffer:

```go
func Int(n *int, buf *Buffer)
func String(s *string, buf *Buffer)
func Bool(b *bool, buf *Buffer)
```

# Buffer with `Writing` flag

The basic building block is a `Buffer` struct that wraps a byte slice, and has a
`Writing` flag.

This flag allows the packing function to fulfill the role of both reading and
writing (serialization and deserialization, or packing and unpacking) at the
same time.

If the buffer has the `Writing` flag set, the packing function _writes_ the
binary representation of the object to the buffer. If the flag is not set, it
_reads_ from the buffer into the object passed.

As a user of this package, you almost never have to worry about checking the
`Writing` flag yourself; the functions for packing the primitive types do that
so that you don't have to.

This architecture solves two problems at once:

- You don't need to duplicate the code for reading and writing.
- You don't need to worry about subtle mistakes with catastrophic effects!

Just pack the fields in order, and you're guaranteed that unpacking will happen
in exactly the same order.

You do need to be careful if you implement a _primitive_ packing function,
where you explicitly check for the `Writing` flag on the buffer, and set the
error flag if some error was encountered.

For example, here's the function for packing a `uint64` using `varint`
encoding:

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

VPack provides enough primitives that you generally don't need to implement your
own, but we don't discourage you from implementing your own primitive packing
functions. You just have to be really careful with them, and test them very
well. Small subtle mistakes here _can_ cause data corruption if you're not
careful!

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

# Versioning

Robust packing for long term storage requires supporting schema evolution
through a version flag: before writing out the fields, write a version number.

In the future, when the struct changes, create a new packing function,
while keeping the old one and changing its content to account for the new
version.

Here's an example of how you would manually manage versioned serialization:

```go
func PackSomething(self *Something, buf *vpack.Buffer) {
    // First, specify the latest version for this type to be written
	var version = 3

	// When the `Writing` flag is not set, this line would set the local
	// variable `version` with the value previously written
    vpack.Int(&version, buf)
    switch version { // check the version we read
        case 1:
			// unpack version 1
			...
		case 2:
			// unpack version 2
			....
		case 3:
			// (un)pack version 3 (the latest)
			....
        default:
            buf.Error = true
    }
}
```

VPack provides a helper function `Versioned` that lets you provide the
serialization function for each version in order

```go
func Versioned[T any](item *T, buf *Buffer, fns ...PackFn[T])
```

Here's how it would simplify version handling code:

```go
func v1PackSomething(self *Something, buf *vpack.Buffer) {
	// deserialize version 1
	...
}

func v2PackSomething(self *Something, buf *vpack.Buffer) {
	// deserialize version 2
	...
}

func v3PackSomething(self *Something, buf *vpack.Buffer) {
	// (de)serialize version 3
	...
}

func PackSomething(self *Something, buf *vpack.Buffer) {
	vpack.Versioned(xyz, buf,
		// list out the version
		v1PackSomething,
		v2PackSomething,
		v3PackSomething,
	)
}
```

Now, how exactly would use versioning to handle schema evolution? The following
is a simplified example to illustrate the basic idea.

Suppose at time t0 your code looked like this:

```go
type XYZ struct {
	Unit int
	Energy int
	Chapter string
	President bool
}

func v1PackXYZ(xyz *XYZ, buf *vpack.Buffer) {
	vpack.Int(&xyz.Unit, buf)
	vpack.Int(&xyz.Energy, buf)
	vpack.String(&xyz.Chapter, buf)
	vpack.Bool(&xyz.President, buf)
}

func PackXYZ(xyz *XYZ, buf *vpack.Buffer) {
	vpack.Versioned(xyz, buf,
		// list out the version
		v1PackXYZ,
	)
}
```

Later at time t1, we remove the `Energy` field, and add a `Price` field.

```go
type XYZ struct {
	Unit int
	Price int
	Chapter string
	President bool
}
```

Now we need to add a new version, and update the code for v1PackXYZ.

An interesting thing to note is now v1PackXYZ will only be called in
deserialization mode!

How do we updated it? We read the fields in the same order as before, but use
local variables for fields that disappeared, and then do some transformation so
that it fits the new format.

What exactly goes into the transformation code is very specific to your
application, so the example here is completely imaginary.

```go
func v1PackXYZ(xyz *XYZ, buf *vpack.Buffer) {
	var energy int // Field that used to exist in previous version

	vpack.Int(&xyz.Unit, buf)
	vpack.Int(&energy, buf) // reading into a local variable
	vpack.String(&xyz.Chapter, buf)
	vpack.Bool(&xyz.President, buf)

	// version migration code:
	// In the new version, price is energy * unit
	xyz.Price = energy * xyz.Unit
}


func v2PackXYZ(xyz *XYZ, buf *vpack.Buffer) {
	vpack.Int(&xyz.Unit, buf)
	vpack.Int(&xyz.Price, buf)
	vpack.String(&xyz.Chapter, buf)
	vpack.Bool(&xyz.President, buf)
}

func PackXYZ(xyz *XYZ, buf *vpack.Buffer) {
	vpack.Versioned(xyz, buf,
		// list out the versions in order
		v1PackXYZ,
		v2PackXYZ,
	)
}
```

You do need to be careful when updating the serialization function for the old
versions so that you don't change the field reading order.
