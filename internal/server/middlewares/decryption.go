package middlewares

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"
	"strings"

	"j30att/observer/internal/encryption"
)

// Decryption decrypts RSA-encrypted request bodies.
func Decryption(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestBody := r.Body
			if requestBody == nil || requestBody == http.NoBody {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(requestBody)
			_ = requestBody.Close()
			if err != nil {
				http.Error(w, "failed to read encrypted request body", http.StatusInternalServerError)
				return
			}
			if len(body) == 0 {
				r.Body = http.NoBody
				r.ContentLength = 0
				next.ServeHTTP(w, r)
				return
			}
			if !hasEncoding(r.Header.Get("Content-Encoding"), encryption.Encoding) {
				http.Error(w, "encrypted request body is required", http.StatusBadRequest)
				return
			}

			body, err = encryption.Decrypt(body, privateKey)
			if err != nil {
				http.Error(w, "failed to decrypt request body", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
			contentEncoding := removeEncoding(r.Header.Get("Content-Encoding"), encryption.Encoding)
			if contentEncoding == "" {
				r.Header.Del("Content-Encoding")
			} else {
				r.Header.Set("Content-Encoding", contentEncoding)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func removeEncoding(header string, encoding string) string {
	values := make([]string, 0)
	for _, value := range strings.Split(header, ",") {
		encodingName, _, _ := strings.Cut(value, ";")
		if strings.EqualFold(strings.TrimSpace(encodingName), encoding) {
			continue
		}
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}

	return strings.Join(values, ", ")
}
