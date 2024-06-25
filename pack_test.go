package vpack

import (
	"testing"
	"time"
	"encoding/json"
)

func toJson(v any) string {
	j, _ := json.Marshal(v)
	return string(j)
}

func _CheckEqual[T comparable](t *testing.T, name string, v1 T, v2 T) {
	t.Logf("Testing %s: %v | %v", name, v1, v2)
	if v1 != v2 {
		t.Errorf("%s not equal: %v != %v", name, v1, v2)
	}
}

func TestPackingThings(t *testing.T) {
	type Other struct {
		I1 int
		S1 string
	}

	PackOther := func(self *Other, buf *Buffer) {
		Int(&self.I1, buf)
		String(&self.S1, buf)
	}

	type Something struct {
		I1 int
		I2 int
		S1 string
		S2 string
		O1 []Other
		B1 bool
		T1 time.Time
	}

	PackSomething := func(self *Something, buf *Buffer) {
		Int(&self.I1, buf)
		Int(&self.I2, buf)
		String(&self.S1, buf)
		StringZ(&self.S2, buf)
		Slice(&self.O1, PackOther, buf)
		Bool(&self.B1, buf)
		UnixTime(&self.T1, buf)
	}

	var obj1 Something
	obj1.I1 = 100 // small number
	obj1.I2 = 43222 // big number
	obj1.S1 = "Hello"
	obj1.S2 = "World"
	obj1.O1 = []Other {
		{I1: 10},
		{S1: "k"},
	}
	obj1.B1 = true
	obj1.T1 = time.UnixMilli(2342000) // packer truncates to seconds

	// encode to bytes and to json
	data := ToBytes(&obj1, PackSomething)
	if data == nil {
		t.Fatal("packing failed")
	}

	obj2 := FromBytes(data, PackSomething)
	if obj2 == nil {
		t.Fatal("unpacking failed")
	}

	json1 := toJson(obj1)
	json2 := toJson(obj2)
	if len(json1) < 4 {
		t.Log(json1)
		t.Fatal("json appears abnormal")
	}
	if json1 != json2 {
		t.Log(json1)
		t.Log(json2)
		t.Fatal("Objects don't appear to match")
	}
}
