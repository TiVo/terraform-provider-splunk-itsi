package util

import (
	"crypto/sha256"
	"encoding/hex"
)

func Sha256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
