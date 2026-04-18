package middlewares_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/middlewares"
)

func TestLoggerWritesRequestAndResponseFields(t *testing.T) {
	var output bytes.Buffer

	oldLogger := log.Logger
	log.Logger = zerolog.New(&output).Level(zerolog.InfoLevel)
	defer func() {
		log.Logger = oldLogger
	}()

	handler := middlewares.Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte("hello"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "hello", rec.Body.String())

	var entry map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
	require.Equal(t, "info", entry["level"])
	require.Equal(t, "handled request", entry["message"])
	require.Equal(t, http.MethodGet, entry["method"])
	require.Equal(t, "/value/gauge/Alloc", entry["uri"])
	require.EqualValues(t, http.StatusCreated, entry["status"])
	require.EqualValues(t, len("hello"), entry["response_size"])
	require.Contains(t, entry, "duration")
}
