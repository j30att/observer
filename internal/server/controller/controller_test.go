package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/repository"
	"j30att/observer/internal/server/router"
)

func TestUpdateMetricHandlerReturnsOK(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController)

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
	r := router.NewRouter(metricController)

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

func TestUpdateMetricHandlerRejectsWrongContentType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController)

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
	r := router.NewRouter(metricController)

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
	r := router.NewRouter(metricController)

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
	r := router.NewRouter(metricController)

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
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewBufferString(`{"id":"Alloc","type":"gauge","value":12.5}`))
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMetricJSONHandlerRejectsMissingValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController)

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
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
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
	r := router.NewRouter(metricController)

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

func TestGetCounterReturnsStoredValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveCounter("PollCount", 3)
	require.NoError(t, err)

	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController)

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
	r := router.NewRouter(metricController)

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
	r := router.NewRouter(metricController)

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
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	require.Contains(t, rec.Body.String(), "Alloc: 12.5")
	require.Contains(t, rec.Body.String(), "PollCount: 3")
}

func newJSONRequest(t *testing.T, method, target string, body any) *http.Request {
	t.Helper()

	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(method, target, bytes.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	return req
}
