package common

import (
	"encoding/base64"
	"fmt"
	"github.com/fatih/structs"
	"net/url"
	"reflect"
	"strings"
)

func Base64StdDecode(s string) (string, error) {
	s = strings.TrimSpace(s)
	saver := s
	if len(s)%4 > 0 {
		s += strings.Repeat("=", 4-len(s)%4)
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return saver, err
	}
	return string(raw), err
}

func BoolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func StringToBool(s string) bool {
	if strings.EqualFold(s, "true") || s == "1" {
		return true
	}
	return false
}

func Base64URLDecode(s string) (string, error) {
	s = strings.TrimSpace(s)
	saver := s
	if len(s)%4 > 0 {
		s += strings.Repeat("=", 4-len(s)%4)
	}
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return saver, err
	}
	return string(raw), err
}

func ObjectToKV(v interface{}, tagName string) (kv []string) {
	a := structs.New(v)
	if tagName != "" {
		a.TagName = tagName
	}
	return MapToKV(a.Map())
}

func MapToKV(m map[string]interface{}) (kv []string) {
	val := reflect.ValueOf(m)
	keys := val.MapKeys()
	for _, k := range keys {
		v := val.MapIndex(k)
		switch v := v.Interface().(type) {
		case map[string]interface{}:
			subKV := MapToKV(v)
			for _, s := range subKV {
				kv = append(kv, fmt.Sprintf("%v.%v", k.String(), s))
			}
		default:
			kv = append(kv, fmt.Sprintf("%v=%v", k.String(), v))
		}
	}
	return kv
}

func StringsToSet(s []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

func StringsMapToSet(s []string, mapper func(s string) string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, v := range s {
		m[mapper(v)] = struct{}{}
	}
	return m
}

func SliceUint64toUint32(from []uint64) (to []uint32) {
	to = make([]uint32, len(from)*2)
	for i := range from {
		to[i*2+1] = uint32(from[i] & 0xffffffff)
		to[i*2] = uint32((from[i] & 0xffffffff00000000) >> 32)
	}
	return to
}

func SetValue(values *url.Values, key string, value string) {
	if value == "" {
		return
	}
	values.Set(key, value)
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MustMapKeys(m interface{}) (keys []string) {
	v := reflect.ValueOf(m)
	vKeys := v.MapKeys()
	for _, k := range vKeys {
		keys = append(keys, k.String())
	}
	return keys
}
