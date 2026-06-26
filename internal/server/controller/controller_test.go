package controller_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/repository"
	"j30att/observer/internal/server/router"
	"j30att/observer/internal/signature"
)

var testLogger = zerolog.Nop()

func TestMetricController(t *testing.T) {
	var (
		repo *repository.InMemoryMetricsRepository
		r    http.Handler
	)

	setup := func(t *testing.T) {
		t.Helper()

		repo = repository.NewMetricsRepository()
		updateMetricCommand := update.New(repo)
		getMetricQuery := get.New(repo)
		listMetricsQuery := getlist.New(repo)
		metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
		r = router.NewRouter(metricController, testLogger, nil)
	}

	t.Run("Тест update handlers", func(t *testing.T) {
		t.Run("Должен вернуть OK для plain update", func(t *testing.T) {
			setup(t)

			req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/12.5", nil)
			req.Header.Set("Content-Type", "text/plain")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Empty(t, rec.Header().Get(signature.Header))
		})

		t.Run("Должен вернуть сохранённую metric для JSON update", func(t *testing.T) {
			setup(t)

			req := newJSONRequest(t, http.MethodPost, "/update", map[string]any{
				"id": "Alloc", "type": "gauge", "value": 12.5,
			})
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var metric map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &metric))
			assert.Equal(t, "Alloc", metric["id"])
			assert.Equal(t, "gauge", metric["type"])
			assert.Equal(t, 12.5, metric["value"])
		})

		t.Run("Должен принять gzip body", func(t *testing.T) {
			setup(t)

			req := newGzipJSONRequest(t, http.MethodPost, "/update", map[string]any{
				"id": "Alloc", "type": "gauge", "value": 12.5,
			})
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			metric, err := repo.Load(context.Background(), "gauge", "Alloc")
			require.NoError(t, err)
			require.NotNil(t, metric.Value)
			assert.Equal(t, 12.5, *metric.Value)
		})

		t.Run("Должен принять batch JSON update", func(t *testing.T) {
			setup(t)

			req := newJSONRequest(t, http.MethodPost, "/updates/", []map[string]any{
				{"id": "Alloc", "type": "gauge", "value": 12.5},
				{"id": "PollCount", "type": "counter", "delta": 2},
			})
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			gauge, err := repo.Load(context.Background(), "gauge", "Alloc")
			require.NoError(t, err)
			require.NotNil(t, gauge.Value)
			assert.Equal(t, 12.5, *gauge.Value)

			counter, err := repo.Load(context.Background(), "counter", "PollCount")
			require.NoError(t, err)
			require.NotNil(t, counter.Delta)
			assert.EqualValues(t, 2, *counter.Delta)
		})

		t.Run("Должен вернуть gzip response", func(t *testing.T) {
			setup(t)

			req := newJSONRequest(t, http.MethodPost, "/update", map[string]any{
				"id": "Alloc", "type": "gauge", "value": 12.5,
			})
			req.Header.Set("Accept-Encoding", "gzip")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
			assert.Contains(t, rec.Header().Values("Vary"), "Accept-Encoding")

			var metric map[string]any
			require.NoError(t, json.Unmarshal(readGzipBody(t, rec.Body.Bytes()), &metric))
			assert.Equal(t, "Alloc", metric["id"])
			assert.Equal(t, "gauge", metric["type"])
			assert.Equal(t, 12.5, metric["value"])
		})

		t.Run("Должен позволить trailing slash", func(t *testing.T) {
			setup(t)

			req := newJSONRequest(t, http.MethodPost, "/update/", map[string]any{
				"id": "PollCount", "type": "counter", "delta": 1,
			})
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		})

		t.Run("Должен отклонить wrong content type для plain update", func(t *testing.T) {
			setup(t)

			req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/12.5", nil)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})

		t.Run("Должен позволить empty content type для plain update", func(t *testing.T) {
			setup(t)

			req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/12.5", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})

		t.Run("Должен отклонить unsupported metric type", func(t *testing.T) {
			setup(t)

			req := httptest.NewRequest(http.MethodPost, "/update/summary/Alloc/12.5", nil)
			req.Header.Set("Content-Type", "text/plain")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), update.ErrUnsupportedMetricType.Error())
		})

		t.Run("Должен отклонить invalid gauge value", func(t *testing.T) {
			setup(t)

			req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/not-a-number", nil)
			req.Header.Set("Content-Type", "text/plain")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})

		t.Run("Должен отклонить wrong content type для JSON update", func(t *testing.T) {
			setup(t)

			req := httptest.NewRequest(
				http.MethodPost,
				"/update",
				bytes.NewBufferString(`{"id":"Alloc","type":"gauge","value":12.5}`),
			)
			req.Header.Set("Content-Type", "text/plain")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), "content type must be application/json, got text/plain")
		})

		t.Run("Должен отклонить missing value", func(t *testing.T) {
			setup(t)

			req := newJSONRequest(t, http.MethodPost, "/update", map[string]any{
				"id": "Alloc", "type": "gauge",
			})
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	})

	t.Run("Тест value handlers", func(t *testing.T) {
		t.Run("Должен вернуть сохранённое gauge value", func(t *testing.T) {
			setup(t)
			require.NoError(t, repo.SaveGauge(context.Background(), "Alloc", 12.5))

			req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "12.5", rec.Body.String())
		})

		t.Run("Должен не сжимать text/plain response", func(t *testing.T) {
			setup(t)
			require.NoError(t, repo.SaveGauge(context.Background(), "Alloc", 12.5))

			req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Empty(t, rec.Header().Get("Content-Encoding"))
			assert.Equal(t, "12.5", rec.Body.String())
		})

		t.Run("Должен вернуть сохранённое JSON value", func(t *testing.T) {
			setup(t)
			require.NoError(t, repo.SaveGauge(context.Background(), "Alloc", 12.5))

			req := newJSONRequest(t, http.MethodPost, "/value", map[string]any{
				"id": "Alloc", "type": "gauge",
			})
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var metric map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &metric))
			assert.Equal(t, "Alloc", metric["id"])
			assert.Equal(t, "gauge", metric["type"])
			assert.Equal(t, 12.5, metric["value"])
		})

		t.Run("Должен вернуть gzip JSON response", func(t *testing.T) {
			setup(t)
			require.NoError(t, repo.SaveGauge(context.Background(), "Alloc", 12.5))

			req := newJSONRequest(t, http.MethodPost, "/value", map[string]any{
				"id": "Alloc", "type": "gauge",
			})
			req.Header.Set("Accept-Encoding", "gzip")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

			var metric map[string]any
			require.NoError(t, json.Unmarshal(readGzipBody(t, rec.Body.Bytes()), &metric))
			assert.Equal(t, "Alloc", metric["id"])
			assert.Equal(t, "gauge", metric["type"])
			assert.Equal(t, 12.5, metric["value"])
		})

		t.Run("Должен позволить trailing slash для JSON value", func(t *testing.T) {
			setup(t)
			require.NoError(t, repo.SaveGauge(context.Background(), "Alloc", 12.5))

			req := newJSONRequest(t, http.MethodPost, "/value/", map[string]any{
				"id": "Alloc", "type": "gauge",
			})
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		})

		t.Run("Должен вернуть сохранённое counter value", func(t *testing.T) {
			setup(t)
			require.NoError(t, repo.SaveCounter(context.Background(), "PollCount", 3))

			req := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "3", rec.Body.String())
		})

		t.Run("Должен вернуть not found для неизвестной plain metric", func(t *testing.T) {
			setup(t)

			req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code)
		})

		t.Run("Должен вернуть not found для неизвестной JSON metric", func(t *testing.T) {
			setup(t)

			req := newJSONRequest(t, http.MethodPost, "/value", map[string]any{
				"id": "Alloc", "type": "gauge",
			})
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code)
		})
	})

	t.Run("Тест list handler", func(t *testing.T) {
		t.Run("Должен вернуть HTML page", func(t *testing.T) {
			setup(t)
			require.NoError(t, repo.SaveGauge(context.Background(), "Alloc", 12.5))
			require.NoError(t, repo.SaveCounter(context.Background(), "PollCount", 3))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
			assert.Contains(t, rec.Body.String(), "Alloc: 12.5")
			assert.Contains(t, rec.Body.String(), "PollCount: 3")
		})

		t.Run("Должен вернуть gzip HTML page", func(t *testing.T) {
			setup(t)
			require.NoError(t, repo.SaveGauge(context.Background(), "Alloc", 12.5))
			require.NoError(t, repo.SaveCounter(context.Background(), "PollCount", 3))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

			body := string(readGzipBody(t, rec.Body.Bytes()))
			assert.Contains(t, body, "Alloc: 12.5")
			assert.Contains(t, body, "PollCount: 3")
		})
	})

	t.Run("Тест gzip middleware", func(t *testing.T) {
		t.Run("Должен отклонить invalid gzip body", func(t *testing.T) {
			setup(t)

			req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewBufferString("not gzip"))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Content-Encoding", "gzip")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	})
}

func newJSONRequest(t *testing.T, method, target string, body any) *http.Request {
	t.Helper()

	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(method, target, bytes.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newGzipJSONRequest(t *testing.T, method, target string, body any) *http.Request {
	t.Helper()

	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err = writer.Write(rawBody)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(method, target, &compressed)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	return req
}

func readGzipBody(t *testing.T, body []byte) []byte {
	t.Helper()

	reader, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer func() {
		_ = reader.Close()
	}()

	rawBody, err := io.ReadAll(reader)
	require.NoError(t, err)

	return rawBody
}
