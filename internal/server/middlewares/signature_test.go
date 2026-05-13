package middlewares

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/signature"
)

func TestSignature(t *testing.T) {
	t.Run("Пропускает запрос если ключ не задан", func(t *testing.T) {
		handler := Signature("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Equal(t, []byte("payload"), body)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewBufferString("payload"))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Пропускает запрос с корректной подписью и восстанавливает body", func(t *testing.T) {
		handler := Signature("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Equal(t, []byte("payload"), body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte(`{"status":"ok"}`))
			require.NoError(t, err)
		}))

		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewBufferString("payload"))
		req.Header.Set(signature.Header, signature.Sign([]byte("payload"), "secret-key"))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Equal(t, signature.Sign(rec.Body.Bytes(), "secret-key"), rec.Header().Get(signature.Header))
	})

	t.Run("Отклоняет запрос с некорректной подписью", func(t *testing.T) {
		handler := Signature("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewBufferString("payload"))
		req.Header.Set(signature.Header, "bad-signature")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, signature.Sign(rec.Body.Bytes(), "secret-key"), rec.Header().Get(signature.Header))
	})

	t.Run("Добавляет подпись ответа если handler не вызывал WriteHeader", func(t *testing.T) {
		handler := Signature("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte("response body"))
			require.NoError(t, err)
		}))

		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewBufferString("payload"))
		req.Header.Set(signature.Header, signature.Sign([]byte("payload"), "secret-key"))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "response body", rec.Body.String())
		assert.Equal(t, signature.Sign(rec.Body.Bytes(), "secret-key"), rec.Header().Get(signature.Header))
	})

	t.Run("Подписывает ответ для запроса без тела", func(t *testing.T) {
		handler := Signature("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte("pong"))
			require.NoError(t, err)
		}))

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set(signature.Header, signature.Sign(nil, "secret-key"))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "pong", rec.Body.String())
		assert.Equal(t, signature.Sign(rec.Body.Bytes(), "secret-key"), rec.Header().Get(signature.Header))
	})
}
