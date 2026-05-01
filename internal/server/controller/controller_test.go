package controller_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/repository"
	"j30att/observer/internal/server/router"
)

var testLogger = zerolog.Nop()

func TestUpdateMetricHandlerReturnsOK(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/12.5", nil)
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateMetricJSONHandlerReturnsStoredMetric(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := newJSONRequest(t, http.MethodPost, "/update", map[string]any{
		"id":    "Alloc",
		"type":  "gauge",
		"value": 12.5,
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var metric map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &metric))
	require.Equal(t, "Alloc", metric["id"])
	require.Equal(t, "gauge", metric["type"])
	require.Equal(t, 12.5, metric["value"])
}

func TestUpdateMetricJSONHandlerAcceptsGzipBody(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := newGzipJSONRequest(t, http.MethodPost, "/update", map[string]any{
		"id":    "Alloc",
		"type":  "gauge",
		"value": 12.5,
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	metric, err := repo.Load("gauge", "Alloc")
	require.NoError(t, err)
	require.NotNil(t, metric.Value)
	require.Equal(t, 12.5, *metric.Value)
}

func TestUpdateMetricJSONHandlerReturnsGzipResponse(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := newJSONRequest(t, http.MethodPost, "/update", map[string]any{
		"id":    "Alloc",
		"type":  "gauge",
		"value": 12.5,
	})
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	require.Contains(t, rec.Header().Values("Vary"), "Accept-Encoding")

	var metric map[string]any
	require.NoError(t, json.Unmarshal(readGzipBody(t, rec.Body.Bytes()), &metric))
	require.Equal(t, "Alloc", metric["id"])
	require.Equal(t, "gauge", metric["type"])
	require.Equal(t, 12.5, metric["value"])
}

func TestUpdateMetricJSONHandlerAllowsTrailingSlash(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := newJSONRequest(t, http.MethodPost, "/update/", map[string]any{
		"id":    "PollCount",
		"type":  "counter",
		"delta": 1,
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestUpdateMetricHandlerRejectsWrongContentType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/12.5", nil)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMetricHandlerAllowsEmptyContentType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/12.5", nil)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateMetricHandlerRejectsUnsupportedMetricType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodPost, "/update/summary/Alloc/12.5", nil)
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), update.ErrUnsupportedMetricType.Error())
}

func TestUpdateMetricHandlerRejectsInvalidGaugeValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/not-a-number", nil)
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMetricJSONHandlerRejectsWrongContentType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewBufferString(`{"id":"Alloc","type":"gauge","value":12.5}`))
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "content type must be application/json, got text/plain")
}

func TestUpdateMetricJSONHandlerRejectsMissingValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := newJSONRequest(t, http.MethodPost, "/update", map[string]any{
		"id":   "Alloc",
		"type": "gauge",
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetMetricReturnsStoredValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "12.5", rec.Body.String())
}

func TestGetMetricDoesNotCompressTextPlainResponse(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Header().Get("Content-Encoding"))
	require.Equal(t, "12.5", rec.Body.String())
}

func TestGetMetricJSONReturnsStoredValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := newJSONRequest(t, http.MethodPost, "/value", map[string]any{
		"id":   "Alloc",
		"type": "gauge",
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var metric map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &metric))
	require.Equal(t, "Alloc", metric["id"])
	require.Equal(t, "gauge", metric["type"])
	require.Equal(t, 12.5, metric["value"])
}

func TestGetMetricJSONReturnsGzipResponse(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := newJSONRequest(t, http.MethodPost, "/value", map[string]any{
		"id":   "Alloc",
		"type": "gauge",
	})
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	var metric map[string]any
	require.NoError(t, json.Unmarshal(readGzipBody(t, rec.Body.Bytes()), &metric))
	require.Equal(t, "Alloc", metric["id"])
	require.Equal(t, "gauge", metric["type"])
	require.Equal(t, 12.5, metric["value"])
}

func TestGetMetricJSONAllowsTrailingSlash(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := newJSONRequest(t, http.MethodPost, "/value/", map[string]any{
		"id":   "Alloc",
		"type": "gauge",
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestGetCounterReturnsStoredValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveCounter("PollCount", 3)
	require.NoError(t, err)

	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "3", rec.Body.String())
}

func TestGetMetricReturnsNotFoundForUnknownMetric(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetMetricJSONReturnsNotFoundForUnknownMetric(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := newJSONRequest(t, http.MethodPost, "/value", map[string]any{
		"id":   "Alloc",
		"type": "gauge",
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestListMetricsReturnsHTMLPage(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	err = repo.SaveCounter("PollCount", 3)
	require.NoError(t, err)

	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	require.Contains(t, rec.Body.String(), "Alloc: 12.5")
	require.Contains(t, rec.Body.String(), "PollCount: 3")
}

func TestListMetricsReturnsGzipHTMLPage(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	err = repo.SaveCounter("PollCount", 3)
	require.NoError(t, err)

	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	body := string(readGzipBody(t, rec.Body.Bytes()))
	require.Contains(t, body, "Alloc: 12.5")
	require.Contains(t, body, "PollCount: 3")
}

func TestGzipMiddlewareRejectsInvalidGzipBody(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController, testLogger)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewBufferString("not gzip"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
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
