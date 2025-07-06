package utils

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrTraversal      = errors.New("traversal failed")
	ErrUnmarshalBytes = errors.New("expected a slice of bytes")
)

func MapSet(m map[string]any, keys []string, val any) (map[string]any, error) {
	curMap := m

	if len(keys) == 0 {
		return m, ErrTraversal
	}

	for _, k := range keys[:len(keys)-1] {
		v, found := curMap[k]
		if found {
			var ok bool
			if curMap, ok = v.(map[string]any); !ok {
				return m, ErrTraversal
			}
		} else {
			nextMap := map[string]any{}
			curMap[k] = nextMap
			curMap = nextMap
		}
	}

	curMap[keys[len(keys)-1]] = val
	return m, nil
}

func MapGet(m any, keys []string) (val any, err error) {
	if len(keys) == 0 {
		err = ErrTraversal
		return
	}
	val = m
	for _, k := range keys {
		asMap, ok := val.(map[string]any)
		if !ok {
			err = ErrTraversal
			return
		}
		val, ok = asMap[k]
		if !ok {
			err = ErrTraversal
			return
		}
	}
	return
}

func Dump(obj any) string {
	objBytes, err := json.Marshal(obj)
	if err != nil {
		objBytes = []byte(fmt.Sprintf("Failed to dump the object as json: %s", err))
	}
	return string(objBytes)
}

func UnmarshalBytes(bytes, value any) error {
	data, ok := bytes.([]byte)
	if !ok {
		return ErrUnmarshalBytes
	}
	return json.Unmarshal(data, value)
}
