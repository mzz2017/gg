package common

import (
	"encoding/base64"
	"fmt"
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
