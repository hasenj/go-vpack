# Introduction

VPack is a general purpose binary data (de)serialization library.

It implements a robust scheme for binary serialization and deserialization of
plain data into and from byte buffers.

While this package was originally designed to serve as a building block to
[VBolt][vbolt], it is in fact general purpose and has no dependency on VBolt.

[vbolt]: https://pkg.go.dev/go.hasen.dev/vbolt

# Serialization Buffer and Mode

The basic building block is a `Buffer` struct that has a backing buffer (a byte
slice) and a Mode, which can be either Serialize or Deserialize.

This allows the "serialization" function to fulfill the role of both reading and
writing (serialization and deserialization) at the same time.

A serialization function takes a pointer to an object (to be (de)serialized) and
a pointer to a Buffer.

If the buffer is in serialization mode, the function writes the binary
representation of the object to the buffer. If the buffer is in deserialization
mode, the function reads from the buffer into the object passed.

As a user of this package, you almost never have to worry about checking the
mode in your own serialization code. At the high level, when you use it to
serialize objects, you are basically just listing the fields you want to
serialize and what function to use to serialize them.

```go
type XYZ struct {
	Unit int
	Energy int
	Chapter string
	President bool
}

func serializeXYZ(xyz *XYZ, buf *store.Buffer) {
	store.Int(&xyz.Unit, buf)
	store.Int(&xyz.Energy, buf)
	store.String(&xyz.Chapter, buf)
	store.Bool(&xyz.President, buf)
}
```

This structure for the serialization API allows binary serialization to be
robust. Without it, you would have to write two separate functions, one to
serialize fields, and another to deserialize them. You would have to be very
very careful about doing the operations in the exact same order! Mistakes would
be very easy to make but very difficult to discover.

By adopting a single function with a mode flag, we solve two problems at once:

- You don't need to write basically the same code twice
- You don't need to worry about making subtle mistakes

This allows the serialization and deserialization of complex objects to be
robust: just serialize the relevant fields in order, and you're guaranteed that
deserialization will happen in exactly the same order.

You do need to be careful if you implement a _primitive_ serialization function,
where you explicitly check for the mode on the buffer, and set the error flag if
some error was encountered.

For example, here's the function for serializing a uint64 with varint encoding:

```go
func VInt64(n *int64, buf *Buffer) {
	switch buf.Mode {
	case Serialize:
		buf.Data = binary.AppendVarint(buf.Data, *n)
	case Deserialize:
		var err error
		*n, err = binary.ReadVarint(buf)
		if err != nil {
			buf.Error = true
		}
	}
}
```

VPack provides enough primitives that you generally don't need to implement your
own, but we don't discourage you from implementing your own primitive
serializations. You just have to be really careful with them, and test them very
well. Small subtle mistakes there can cause data corruption if you're not
careful!.

# Composite objects

VPack provides serialization functions for slices and maps. They don't fullfil
the serialization function signature, but you can use them inside the
serialization function for your types.

```go
func Slice[T any](list *[]T, fn SerializeFn[T], buf *Buffer)
func Map[K comparable, T any](m *map[K]T, keyFn SerializeFn[K], valFn SerializeFn[T], buf *Buffer)
```

To serialize an int list field `[]int` named `Ids`, you can call:

```go
vpack.Slice(&item.Ids, vpack.Int, buf)
```

You can also use to serialize slices of objects. Support you have these types:

```go
type ArticleAuthor struct {
	AuthorId int
	Nickname string
}

type Article struct {
	....
	Authors []ArticleAuthor
}

func SerializeArticleAuthor(self *ArticleAuthor, buf *vpack.Buffer) {
	....
}

func SerializeArticle(self *Article, buf *vpack.Buffer) {
	....
	vpack.Slice(&self.Authors, SerializeArticleAuthor, buf)
}
```

# Versioning

Robust serialization for long term storage requires supporting schema evolution
through a version flag: before writing out the fields, write a version number.

In the future, when the struct changes, create a new serialization function,
while keeping the old one and changing its content to account for the new
version.

Here's an example of how you would manually manage versioned serialization:

```go
func SerializeSomething(self *Something, buf *store.Buffer) {
    var version = 1
    store.Int(&version, buf)
    switch version {
        case 1:
			// deserialize version 1
			...
		case 2:
			// deserialize version 2
			....
		case 3:
			// (de)serialize version 3 (the latest)
			....
        default:
            buf.Error = true
    }
}
```

VPack provides a helper function `Versioned` that lets you provide the
serialization function for each version in order

```go
func Versioned[T any](item *T, buf *Buffer, fns ...SerializeFn[T])
```

Here's how it would simplify version handling code:

```go
func v1SerializeSomething(self *Something, buf *store.Buffer) {
	// deserialize version 1
	...
}

func v2SerializeSomething(self *Something, buf *store.Buffer) {
	// deserialize version 2
	...
}

func v3SerializeSomething(self *Something, buf *store.Buffer) {
	// (de)serialize version 3
	...
}


func SerializeSomething(self *Something, buf *store.Buffer) {
	store.Versioned(xyz, buf,
		// list out the version
		v1SerializeSomething,
		v2SerializeSomething,
		v3SerializeSomething,
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

	func v1SerializeXYZ(xyz *XYZ, buf *store.Buffer) {
	    store.Int(&xyz.Unit, buf)
	    store.Int(&xyz.Energy, buf)
	    store.String(&xyz.Chapter, buf)
	    store.Bool(&xyz.President, buf)
	}

	func SerializeXYZ(xyz *XYZ, buf *store.Buffer) {
	    store.Versioned(xyz, buf,
	        // list out the version
	        v1SerializeXYZ,
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

Now we need to add a new version, and update the code for v1SerializeXYZ.

An interesting thing to note is now v1SerializeXYZ will only be called in
deserialization mode!

How do we updated it? We read the fields in the same order as before, but use
local variables for fields that disappeared, and then do some transformation so
that it fits the new format.

What exactly goes into the transformation code is very specific to your
application, so the example here is completely imaginary.

```go
func v1SerializeXYZ(xyz *XYZ, buf *store.Buffer) {
	var energy int // Field that used to exist in previous version

	store.Int(&xyz.Unit, buf)
	store.Int(&energy, buf) // reading into a local variable
	store.String(&xyz.Chapter, buf)
	store.Bool(&xyz.President, buf)

	// version migration code:
	// In the new version, price is energy * unit
	xyz.Price = energy * xyz.Unit
}


func v2SerializeXYZ(xyz *XYZ, buf *store.Buffer) {
	store.Int(&xyz.Unit, buf)
	store.Int(&xyz.Price, buf)
	store.String(&xyz.Chapter, buf)
	store.Bool(&xyz.President, buf)
}

func SerializeXYZ(xyz *XYZ, buf *store.Buffer) {
	store.Versioned(xyz, buf,
		// list out the versions in order
		v1SerializeXYZ,
		v2SerializeXYZ,
	)
}
```

You do need to be careful when updating the serialization function for the old
versions so that you don't change the field reading order.
