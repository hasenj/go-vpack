package vpack

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
)

const UUID_SIZE = 16

// UUID is a continous buffer of 16 bytes
type UUID [UUID_SIZE]byte

// GenerateUUID creates a UUID (a 16 byte buffer) and fills it with bytes from the
// cryptographically secure random generator provided by the OS
func GenerateUUID() UUID {
	var id UUID
	rand.Read(id[:])
	return id
}

// PackUUID is the serializer/deserializer function for UUID
func PackUUID(id *UUID, buf *Buffer) {
	for i := range id {
		bptr := &((*id)[i])
		Byte(bptr, buf)
	}
}

var rawUrlEnc = base64.RawURLEncoding

// String returns a url-safe base64 representation of the UUID bytes.
func (u UUID) String() string {
	return rawUrlEnc.EncodeToString(u[:])
}

var InvalidUUIDSize = errors.New("InvalidUUIDSize")

func (u *UUID) FromString(suuid string) error {
	buf, err := rawUrlEnc.DecodeString(suuid)
	if err != nil {
		return err
	}
	if len(buf) != UUID_SIZE {
		return InvalidUUIDSize
	}
	copy((*u)[:], buf)
	return nil
}

/*
	NOTE: this is important: the json marshaler should *not* be pointer based,
	but the unmarshaler *must* be pointer based
*/

// MarshalJSON implements the json marshalling interface for UUID
func (u UUID) MarshalJSON() ([]byte, error) {
	strValue := rawUrlEnc.EncodeToString(u[:])
	return json.Marshal(strValue)
}

// UnmarshalJSON implements the json marshalling interface for UUID
func (u *UUID) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(raw, []byte("null")) {
		return nil // no error
	}
	var strValue string
	var err error
	err = json.Unmarshal(raw, &strValue)
	if err != nil {
		return err
	}
	if len(strValue) == 0 {
		return nil
	}
	return u.FromString(strValue)
}
