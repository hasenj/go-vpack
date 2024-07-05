package vpack

import (
	"encoding/binary"
	"math"
)

var BigEndian = binary.BigEndian

// FUInt64 implements fixed size serialization of uint64. It writes data in big
// endian, making it suitable for int keys to bolt.
func FUInt64(n *uint64, buf *Buffer) {
	if buf.Writing {
		buf.Data = BigEndian.AppendUint64(buf.Data, *n)
	} else {
		slice := buf.ReadBytes(8)
		*n = BigEndian.Uint64(slice)
	}
}

// FUInt32 implements fixed size serialization of uint32. It writes data in big
// endian, making it suitable for int keys to bolt.
func FUInt32(n *uint32, buf *Buffer) {
	if buf.Writing {
		buf.Data = BigEndian.AppendUint32(buf.Data, *n)
	} else {
		slice := buf.ReadBytes(2)
		*n = BigEndian.Uint32(slice)
	}
}

// FUInt16 implements fixed size serialization of uint16. It writes data in big
// endian, making it suitable for int keys to bolt.
func FUInt16(n *uint16, buf *Buffer) {
	if buf.Writing {
		buf.Data = BigEndian.AppendUint16(buf.Data, *n)
	} else {
		slice := buf.ReadBytes(2)
		*n = BigEndian.Uint16(slice)
	}
}

// FInt64 implements fixed size serialization of int64. It writes data in big
// endian, making it suitable for int keys to bolt.
func FInt64(n *int64, buf *Buffer) {
	var u = uint64(*n)
	FUInt64(&u, buf)
	*n = int64(u)
}

// FInt implements fixed size serialization of int (as 64 bits). It writes data
// in big endian, making it suitable for int keys to bolt.
func FInt(n *int, buf *Buffer) {
	var u = uint64(*n)
	FUInt64(&u, buf)
	*n = int(u)
}

func Float64(n *float64, buf *Buffer) {
	// Flaot64bit and Float64frombits are just transmute casts that should cost nothing
	var u = math.Float64bits(*n)
	FUInt64(&u, buf)
	*n = math.Float64frombits(u)
}

/*
	switch buf.Mode {
	case Serialize:
	case Deserialize:
	}
*/

// Byte implements serialization for a single byte
func Byte(b *byte, buf *Buffer) {
	if buf.Writing {
		buf.WriteBytes(*b)
	} else {
		*b = buf.ReadBytes(1)[0]
	}
}

// Bool implements serialization for a bool
func Bool(b *bool, buf *Buffer) {
	var bt byte
	if *b {
		bt = 1
	}
	Byte(&bt, buf)
	*b = bt != 0
}

// VInt64 implements varint encoding for int64. Varint users fewer bytes for
// small values.
func VInt64(n *int64, buf *Buffer) {
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

// VUInt64 implements varint encoding for uin64. Varint users fewer bytes for
// small values.
func VUInt64(n *uint64, buf *Buffer) {
	if buf.Writing {
		buf.Data = binary.AppendUvarint(buf.Data, *n)
	} else {
		var err error
		*n, err = binary.ReadUvarint(buf)
		if err != nil {
			buf.Error = true
		}
	}
}

// Int implements varint encoding for int (as int64). Varint users fewer bytes for
// small values.
func Int(n *int, buf *Buffer) {
	var n64 = int64(*n)
	VInt64(&n64, buf)
	*n = int(n64)
}

// UInt implements varint encoding for uint (as uint64). Varint users fewer
// bytes for small values.
func UInt(n *uint, buf *Buffer) {
	var n64 = uint64(*n)
	VUInt64(&n64, buf)
	*n = uint(n64)
}

type IntBased interface {
	~int | ~int64
}

// IntEnum implements varint encoding for an int (or int64) based enum types
func IntEnum[T IntBased](n *T, buf *Buffer) {
	var n64 = int64(*n)
	VInt64(&n64, buf)
	*n = T(n64)
}

// Rune implements serialization for a single rune as a varint.
func Rune(r *rune, buf *Buffer) {
	var n64 = int64(*r)
	VInt64(&n64, buf)
	*r = rune(n64)
}

func Version(max int, buf *Buffer) int {
	var v = max
	Int(&v, buf)
	if v > max {
		buf.Error = true
	}
	return v
}
