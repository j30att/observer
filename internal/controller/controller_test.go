package controller_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"j30att/observer/internal/commands"
	"j30att/observer/internal/controller"
	"j30att/observer/internal/repository"
	"j30att/observer/internal/router"
)

func TestUpdateMetricHandlerReturnsOK(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := commands.NewUpdateMetricCommand(repo)
	metricController := controller.NewMetricController(updateMetricCommand)
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/12.5", nil)
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestUpdateMetricHandlerRejectsWrongContentType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := commands.NewUpdateMetricCommand(repo)
	metricController := controller.NewMetricController(updateMetricCommand)
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/12.5", nil)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUpdateMetricHandlerRejectsUnsupportedMetricType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := commands.NewUpdateMetricCommand(repo)
	metricController := controller.NewMetricController(updateMetricCommand)
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodPost, "/update/summary/Alloc/12.5", nil)
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), commands.ErrUnsupportedMetricType.Error()) {
		t.Fatalf("expected body to contain %q, got %q", commands.ErrUnsupportedMetricType.Error(), rec.Body.String())
	}
}

func TestUpdateMetricHandlerRejectsInvalidGaugeValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := commands.NewUpdateMetricCommand(repo)
	metricController := controller.NewMetricController(updateMetricCommand)
	r := router.NewRouter(metricController)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/not-a-number", nil)
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
