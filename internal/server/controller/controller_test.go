package controller_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/handler"
	"j30att/observer/internal/server/repository"
	"j30att/observer/internal/server/router"
)

func TestUpdateMetricHandlerReturnsOK(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := handler.NewUpdateMetricHandler(repo)
	metricController := controller.NewMetricController(updateMetricCommand)
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/12.5", nil)
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateMetricHandlerRejectsWrongContentType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := handler.NewUpdateMetricHandler(repo)
	metricController := controller.NewMetricController(updateMetricCommand)
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/12.5", nil)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMetricHandlerRejectsUnsupportedMetricType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := handler.NewUpdateMetricHandler(repo)
	metricController := controller.NewMetricController(updateMetricCommand)
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodPost, "/update/summary/Alloc/12.5", nil)
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), handler.ErrUnsupportedMetricType.Error())
}

func TestUpdateMetricHandlerRejectsInvalidGaugeValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := handler.NewUpdateMetricHandler(repo)
	metricController := controller.NewMetricController(updateMetricCommand)
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/not-a-number", nil)
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
