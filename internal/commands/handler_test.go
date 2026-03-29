package commands_test

import (
	"errors"
	"testing"

	"j30att/observer/internal/commands"
	"j30att/observer/internal/model"
	"j30att/observer/internal/repository"
)

func TestExecuteSavesGauge(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := commands.NewUpdateMetricCommand(repo)

	err := command.Execute(model.Gauge, "Alloc", "12.5")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	metric, err := repo.Load(model.Gauge, "Alloc")
	if err != nil {
		t.Fatalf("expected stored gauge, got %v", err)
	}

	if metric.Value == nil || *metric.Value != 12.5 {
		t.Fatalf("expected gauge value 12.5, got %+v", metric.Value)
	}
}

func TestExecuteAccumulatesCounter(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := commands.NewUpdateMetricCommand(repo)

	err := command.Execute(model.Counter, "PollCount", "2")
	if err != nil {
		t.Fatalf("expected no error on first update, got %v", err)
	}

	err = command.Execute(model.Counter, "PollCount", "3")
	if err != nil {
		t.Fatalf("expected no error on second update, got %v", err)
	}

	metric, err := repo.Load(model.Counter, "PollCount")
	if err != nil {
		t.Fatalf("expected stored counter, got %v", err)
	}

	if metric.Delta == nil || *metric.Delta != 5 {
		t.Fatalf("expected counter delta 5, got %+v", metric.Delta)
	}
}

func TestExecuteReturnsErrorForUnsupportedMetricType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := commands.NewUpdateMetricCommand(repo)

	err := command.Execute("summary", "Alloc", "12.5")
	if !errors.Is(err, commands.ErrUnsupportedMetricType) {
		t.Fatalf("expected ErrUnsupportedMetricType, got %v", err)
	}
}

func TestExecuteReturnsErrorForInvalidCounterValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := commands.NewUpdateMetricCommand(repo)

	err := command.Execute(model.Counter, "PollCount", "abc")
	if err == nil {
		t.Fatal("expected error for invalid counter value, got nil")
	}
}
