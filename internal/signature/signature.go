package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

const Header = "HashSHA256"

func Sign(data []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func Verify(data []byte, key string, signature string) bool {
	expected := Sign(data, key)
	return hmac.Equal([]byte(signature), []byte(expected))
}
