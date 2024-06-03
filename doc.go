/*
Package store implements a robust scheme for binary serialization and
deserialization of plain data into and from byte buffers.

The purpose of this package is to serve as the serialization building block to
the boltdb-backed storage engine, but it's general purpose enough that it can be
used in any other context, such as serialization to/from files.

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
mode in your own serialization code. At the high level, you just list the fields
you want to serialize and what function to use to serialize them.

This allows the serialization and deserialization of complex objects to be
robust: just serialize the relevant fields in order, and you're guaranteed that
deserialization will happen in exactly the same order.

# Versioning

Robust serialization for long term storage requires supporting schema evolution
through a version flag: before writing out the fields, write a version number.

In the future, when the struct changes, create a new serialization function,
while keeping the old one and changing its content to account for the new
version.

Example:

At time t0:

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

Later at time t1, we remove the `Energy` field, and add a `Price` field.

	type XYZ struct {
	    Unit int
	    Price int
	    Chapter string
	    President bool
	}

	// v1SerializeXYZ needs to be modified to accomodate the new structure,
	// while still reading the fields in the same order

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
*/
package store
