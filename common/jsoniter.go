package common

import (
	"encoding/json"
	jsoniter "github.com/json-iterator/go"
	"unsafe"
)

type FuzzyStringDecoder struct {
}

func (decoder *FuzzyStringDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	valueType := iter.WhatIsNext()
	switch valueType {
	case jsoniter.NumberValue:
		var number json.Number
		iter.ReadVal(&number)
		*((*string)(ptr)) = string(number)
	case jsoniter.StringValue:
		*((*string)(ptr)) = iter.ReadString()
	case jsoniter.NilValue:
		iter.Skip()
		*((*string)(ptr)) = ""
	default:
		iter.ReportError("FuzzyStringDecoder", "not number or string")
	}
}
