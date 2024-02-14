package crypt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func SignDataWithSHA256(plainText []byte, key string) (string, error) {
	if key != "" {
		h := hmac.New(sha256.New, []byte(key))
		h.Write(plainText)
		s := h.Sum(nil)
		h.Reset()
		return hex.EncodeToString(s), nil
	}
	return "", fmt.Errorf("key parameter is empty")
}

func CheckHashSHA256(resultHash, reqHash string) bool {
	return hmac.Equal([]byte(resultHash), []byte(reqHash))
}
