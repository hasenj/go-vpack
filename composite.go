package vpack

import (
	"encoding"
	"time"

	"go.hasen.dev/generic"
)

// String implements serialization for a string by first writing out the length
// in bytes (as a varint) then dumping the actual bytes into the buffer. When
// deserializing, it starts by reading the length (as a varint) then taking a
// slice of the input buffer and cloning it to a string
func String(s *string, buf *Buffer) {
	var size = len(*s)
	Int(&size, buf)
	if buf.Writing {
		var pos = len(buf.Data)
		buf.EnsureSpace(size)
		copy(buf.Data[pos:pos+size], *s)
	} else {
		// ReadBytes generally returns a slice into the buffer, not a copy of the data
		// But `string(...)` copies the data to a new buffer in memory, so we should be ok!
		*s = string(buf.ReadBytes(size))
	}
}

// StringZ implement serialization for a string using null-byte termination.
// This allows is to be used in the key of a boltdb key
func StringZ(s *string, buf *Buffer) {
	if buf.Writing {
		var pos = len(buf.Data)
		var size = len(*s)
		buf.EnsureSpace(size)
		copy(buf.Data[pos:pos+size], *s)
		buf.WriteBytes(0)
	} else {
		var start = buf.Pos
		var end = start
		for end < len(buf.Data) && buf.Data[end] != 0 {
			end++
		}
		buf.Pos = end + 1
		*s = string(buf.Data[start:end])
	}
}

// ByteSlice implements serialization for a byte slice. It's more or less just
// like String.
func ByteSlice(s *[]byte, buf *Buffer) {
	var size = len(*s)
	Int(&size, buf)
	if buf.Writing {
		var pos = len(buf.Data)
		buf.EnsureSpace(size)
		copy(buf.Data[pos:pos+size], *s)
	} else {
		// ReadBytes generally returns a slice into the buffer, not a copy of the data
		// we need to copy it out
		*s = make([]byte, size)
		copy(*s, buf.ReadBytes(size))
	}
}

/*
// unfortunately not possible with the current generics system :(
func ByteArray[N int](s *[N]byte, buf *Buffer) {
	switch buf.Mode {
	case Serialize:
		buf.Data = append(buf.Data, *s[:])
	case Deserialize:
		// ReadBytes generally returns a slice into the buffer, not a copy of the data
		// we need to copy it out
		copy(*s, buf.ReadBytes(len(*s)))
	}
}
*/

// Slice is a helper for serialization a slice of some type, given its
// serialization function. It starts by reading/writing the length of the slice,
// then uses the provided serialization function to serialize each individual
// item in the slice.
func Slice[T any](list *[]T, fn PackFn[T], buf *Buffer) {
	var size = len(*list)
	Int(&size, buf)
	if !buf.Writing {
		*list = make([]T, size)
	}
	for index := range *list {
		var item = &(*list)[index]
		fn(item, buf)
	}
}

func Map[K comparable, T any](m *map[K]T, keyFn PackFn[K], valFn PackFn[T], buf *Buffer) {
	var size = len(*m)
	Int(&size, buf)
	if buf.Writing {
		for key, val := range *m {
			keyFn(&key, buf)
			valFn(&val, buf)
		}
	} else {
		generic.InitMap(m)
		for i := 0; i < size; i++ {
			var key K
			var val T
			keyFn(&key, buf)
			valFn(&val, buf)
			(*m)[key] = val
		}
	}
}

type Binary interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// BinaryMarshal implements serialization for an object that implements the
// BinaryMarshaler and BinaryUnmarshaler interfaces from the standard library.
func BinaryMarshal(b Binary, buf *Buffer) {
	if buf.Writing {
		var data, err = b.MarshalBinary()
		if err != nil {
			buf.Error = true
			return
		}
		ByteSlice(&data, buf)
	} else {
		var data []byte
		ByteSlice(&data, buf)
		var err = b.UnmarshalBinary(data)
		if err != nil {
			buf.Error = true
			return
		}
	}
}

// Time implement serialization for the std library's Time object using the
// Binary Marshalling interface
func Time(t *time.Time, buf *Buffer) {
	BinaryMarshal(t, buf)
}

// UnixTime serializes Time as a unix timestamp, in other wrods, the resolution
// is truncated to the seconds level, and the location data is omitted. It also
// uses variable encoding to take as little space as possible.
//
// It can store a reasonably accurate timestamp in 5 or 6 bytes.
//
// If you require subsecond accuracy, don't use this function.
func UnixTime(t *time.Time, buf *Buffer) {
	var seconds int64

	if buf.Writing {
		seconds = t.Unix()
	}

	VInt64(&seconds, buf)

	if !buf.Writing {
		*t = time.Unix(seconds, 0)
	}
}

// UnixTimeKey is similar to UnixTime, but uses fixed encoding so the value is
// suitable for a bucket key so we can iterate by timestamp
//
// If you require subsecond accuracy, don't use this function.
func UnixTimeKey(t *time.Time, buf *Buffer) {
	var seconds int64

	if buf.Writing {
		seconds = t.Unix()
	}

	FInt64(&seconds, buf)

	if !buf.Writing {
		*t = time.Unix(seconds, 0)
	}
}

// UnixTimeMilli is similar to UnixTime but truncates to the MilliSecond level
// making it more suitable for cases where sub-second accuracy is required
func UnixTimeMilli(t *time.Time, buf *Buffer) {
	var ms int64

	if buf.Writing {
		ms = t.UnixMilli()
	}

	VInt64(&ms, buf)

	if !buf.Writing {
		*t = time.UnixMilli(ms)
	}
}

// UnixTimeMilliKey is similar to UnixTimeMilli, but uses fixed encoding so the
// value is suitable for a bucket key so we can iterate by timestamp
func UnixTimeMilliKey(t *time.Time, buf *Buffer) {
	var ms int64

	if buf.Writing {
		ms = t.UnixMilli()
	}

	FInt64(&ms, buf)

	if !buf.Writing {
		*t = time.UnixMilli(ms)
	}
}
