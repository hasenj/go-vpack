package store

import "io"

type Mode int

const (
	Serialize Mode = iota
	Deserialize
)

// Buffer is a byte buffer used to serialize data into, or
// deserialize data from, depending on the mode.
type Buffer struct {
	Data  []byte
	Pos   int  // reading position; not used for writing
	Error bool // TODO: something better than this to report errors with more info?
	Mode  Mode
}

// NewReader prepares a Buffer for deserializing data from
// the backing byte buffer. The caller owns the data.
func NewReader(data []byte) *Buffer {
	return &Buffer{
		Data: data,
		Mode: Deserialize,
	}
}

// NewWriter prepares a buffer for serializing data into.
// The backing buffer is owned by Buffer, but when
// serialization is done, the caller may use it.
func NewWriter() *Buffer {
	return &Buffer{
		Data: make([]byte, 0, 64),
		Mode: Serialize,
	}
}

func (b *Buffer) ReadingDone() bool {
	return b.Pos >= len(b.Data)
}

// Ensure there's at least n bytes in the buffer starting from current position
func (b *Buffer) EnsureSpace(n int) {
	var desiredSize = len(b.Data) + n
	if len(b.Data) < desiredSize {
		if cap(b.Data) >= desiredSize {
			b.Data = b.Data[:desiredSize]
		} else {
			b.Data = append(b.Data, make([]byte, desiredSize-len(b.Data))...)
		}
	}
}

func (b *Buffer) WriteBytes(newData ...byte) {
	b.Data = append(b.Data, newData...)
}

// implements io.ByteReader
func (buf *Buffer) ReadByte() (b byte, err error) {
	if buf.Pos >= len(buf.Data) {
		err = io.EOF
		return
	}
	b = buf.Data[buf.Pos]
	buf.Pos += 1
	return
}

// ReadBytes does not expand the buffer to fit the required size.
// Instead, if there's no enough size, it sets the error flag.
func (b *Buffer) ReadBytes(n int) []byte {
	length := len(b.Data)
	if b.Pos+n > length { // unhappy case
		remaining := b.Data[b.Pos:]
		result := make([]byte, n)
		copy(result, remaining)
		b.Pos = length
		b.Error = true
		return result
	}
	start := b.Pos // superfluous var for readability?
	end := b.Pos + n
	result := b.Data[start:end]
	b.Pos = end
	return result
}
