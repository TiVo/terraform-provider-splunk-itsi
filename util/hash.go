package util

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func Sha256[T any](data T) string {
	var bytes []byte
	var err error

	switch any(data).(type) {
	case string:
		bytes = []byte(any(data).(string))
	case []byte:
		bytes = any(data).([]byte)
	default:
		bytes, err = json.Marshal(data)
		if err != nil {
			panic(fmt.Sprintf("Sha256 JSON marshalling error: %#v", data))
		}
	}

	h := sha256.New()
	h.Write(bytes)
	return hex.EncodeToString(h.Sum(nil))
}
