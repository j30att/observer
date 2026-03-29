package repository_test

import (
	"errors"
	"testing"

	"j30att/observer/internal/model"
	"j30att/observer/internal/repository"
)

func TestSaveGaugeAndLoadReturnsStoredValue(t *testing.T) {
	repo := repository.NewMetricsRepository()

	err := repo.SaveGauge("Alloc", 12.5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	metric, err := repo.Load(model.Gauge, "Alloc")
	if err != nil {
		t.Fatalf("expected stored metric, got %v", err)
	}

	if metric.ID != "Alloc" {
		t.Fatalf("expected metric ID %q, got %q", "Alloc", metric.ID)
	}

	if metric.MType != model.Gauge {
		t.Fatalf("expected metric type %q, got %q", model.Gauge, metric.MType)
	}

	if metric.Value == nil || *metric.Value != 12.5 {
		t.Fatalf("expected gauge value 12.5, got %+v", metric.Value)
	}
}

func TestSaveCounterAccumulatesValue(t *testing.T) {
	repo := repository.NewMetricsRepository()

	err := repo.SaveCounter("PollCount", 2)
	if err != nil {
		t.Fatalf("expected no error on first save, got %v", err)
	}

	err = repo.SaveCounter("PollCount", 3)
	if err != nil {
		t.Fatalf("expected no error on second save, got %v", err)
	}

	metric, err := repo.Load(model.Counter, "PollCount")
	if err != nil {
		t.Fatalf("expected stored metric, got %v", err)
	}

	if metric.Delta == nil || *metric.Delta != 5 {
		t.Fatalf("expected counter delta 5, got %+v", metric.Delta)
	}
}

func TestLoadReturnsErrMetricNotFoundForMissingGauge(t *testing.T) {
	repo := repository.NewMetricsRepository()

	_, err := repo.Load(model.Gauge, "UnknownMetric")
	if !errors.Is(err, repository.ErrMetricNotFound) {
		t.Fatalf("expected ErrMetricNotFound, got %v", err)
	}
}

func TestLoadReturnsErrMetricNotFoundForUnsupportedType(t *testing.T) {
	repo := repository.NewMetricsRepository()

	_, err := repo.Load("summary", "Alloc")
	if !errors.Is(err, repository.ErrMetricNotFound) {
		t.Fatalf("expected ErrMetricNotFound, got %v", err)
	}
}
