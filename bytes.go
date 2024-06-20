package vpack

import "errors"

// TODO: test the shit out of this module!!

var GenericError = errors.New("Deserialization error")

// SerializeFn is a generic serialization function that can be used either to
// serialize or deserialize data, depending on the buffer's mode.
type SerializeFn[T any] func(data *T, buffer *Buffer)

func ToBytes[T any](obj *T, fn SerializeFn[T]) []byte {
	buf := NewWriter()
	fn(obj, buf)
	if buf.Error {
		return nil
	} else {
		return buf.Data
	}
}

func FromBytes[T any](data []byte, fn SerializeFn[T]) *T {
	var obj T
	if FromBytesInto(data, &obj, fn) {
		return &obj
	} else {
		return nil
	}
}

func FromBytesInto[T any](data []byte, obj *T, fn SerializeFn[T]) bool {
	buf := NewReader(data)
	fn(obj, buf)
	return !buf.Error
}
