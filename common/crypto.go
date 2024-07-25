package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sha256HMAC(msg string, key []byte) (string, error) {
	h := hmac.New(sha256.New, key)
	if _, err := h.Write([]byte(msg)); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
