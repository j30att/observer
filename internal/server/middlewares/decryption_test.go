package middlewares_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/compression"
	"j30att/observer/internal/encryption"
	"j30att/observer/internal/server/middlewares"
	"j30att/observer/internal/signature"
)

func TestDecryption(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("Должен выполниться без ошибок", func(t *testing.T) {
		payload := []byte(`{"id":"Alloc","type":"gauge","value":12.5}`)
		compressed, err := compression.CompressGzip(payload)
		require.NoError(t, err)
		encrypted, err := encryption.Encrypt(compressed, &privateKey.PublicKey)
		require.NoError(t, err)

		var received []byte
		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received, err = io.ReadAll(r.Body)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
		})
		handler := middlewares.Signature("secret-key")(
			middlewares.Decryption(privateKey)(middlewares.Gzip(finalHandler)),
		)
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(encrypted))
		req.Header.Set("Content-Encoding", "gzip, rsa")
		req.Header.Set(signature.Header, signature.Sign(encrypted, "secret-key"))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, payload, received)
	})

	t.Run("Ошибка, поврежденное зашифрованное тело", func(t *testing.T) {
		handler := middlewares.Decryption(privateKey)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Encoding", encryption.Encoding)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Ошибка, незашифрованное непустое тело", func(t *testing.T) {
		handler := middlewares.Decryption(privateKey)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader([]byte("plain text")))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Должен пропустить запрос с пустым телом", func(t *testing.T) {
		handler := middlewares.Decryption(privateKey)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}
