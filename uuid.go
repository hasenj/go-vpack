package store

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
)

const UUID_SIZE = 16

type UUID [UUID_SIZE]byte

func GenerateUUID() UUID {
	var id UUID
	rand.Read(id[:])
	return id
}

func SerializeUUID(id *UUID, buf *Buffer) {
	for i := range id {
		bptr := &((*id)[i])
		Byte(bptr, buf)
	}
}

var rawUrlEnc = base64.RawURLEncoding

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

// NOTE: this is important: the json marshaler should *not* be pointer based,
// but the unmarshaler *must* be pointer based
func (u UUID) MarshalJSON() ([]byte, error) {
	strValue := rawUrlEnc.EncodeToString(u[:])
	return json.Marshal(strValue)
}

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
