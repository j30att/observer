package middlewares_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/middlewares"
)

func TestLoggerMiddleware(t *testing.T) {
	var (
		output  bytes.Buffer
		handler http.Handler
	)

	setup := func(t *testing.T, next http.HandlerFunc) {
		t.Helper()

		output.Reset()
		logger := zerolog.New(&output).Level(zerolog.InfoLevel)
		handler = middlewares.Logger(logger)(next)
	}

	t.Run("Тест middleware Logger", func(t *testing.T) {
		t.Run("Должен записать поля request и response", func(t *testing.T) {
			setup(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, err := w.Write([]byte("hello"))
				require.NoError(t, err)
			})

			req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusCreated, rec.Code)
			assert.Equal(t, "hello", rec.Body.String())

			var entry map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
			assert.Equal(t, "info", entry["level"])
			assert.Equal(t, "handled request", entry["message"])
			assert.Equal(t, http.MethodGet, entry["method"])
			assert.Equal(t, "/value/gauge/Alloc", entry["uri"])
			assert.EqualValues(t, http.StatusCreated, entry["status"])
			assert.EqualValues(t, len("hello"), entry["response_size"])
			assert.Contains(t, entry, "duration")
		})

		t.Run("Должен записать OK если handler не вызвал WriteHeader явно", func(t *testing.T) {
			setup(t, func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte("hello"))
				require.NoError(t, err)
			})

			req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			var entry map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
			assert.EqualValues(t, http.StatusOK, entry["status"])
			assert.EqualValues(t, len("hello"), entry["response_size"])
		})
	})
}
