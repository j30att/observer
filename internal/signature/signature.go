package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Header is the HTTP header carrying the HMAC-SHA256 signature.
const Header = "HashSHA256"

// Sign returns a hex-encoded HMAC-SHA256 signature for data.
func Sign(data []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify reports whether signature matches data for key.
func Verify(data []byte, key string, signature string) bool {
	expected := Sign(data, key)
	return hmac.Equal([]byte(signature), []byte(expected))
}
