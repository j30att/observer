package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"j30att/observer/internal/agent/repository"
)

func TestMetricsRepository(t *testing.T) {
	var repo *repository.MetricsRepository

	setup := func(t *testing.T) {
		t.Helper()

		repo = repository.NewMetricsRepository()
	}

	t.Run("Тест метода SaveGauge", func(t *testing.T) {
		t.Run("Должен хранить последнее значение gauge", func(t *testing.T) {
			setup(t)

			repo.SaveGauge("Alloc", 10.5)
			repo.SaveGauge("Alloc", 12.5)

			snapshot := repo.Snapshot()

			assert.Equal(t, 12.5, snapshot.Gauges["Alloc"])
		})
	})

	t.Run("Тест метода SaveCounter", func(t *testing.T) {
		t.Run("Должен накапливать counter", func(t *testing.T) {
			setup(t)

			repo.SaveCounter("PollCount", 1)
			repo.SaveCounter("PollCount", 1)

			snapshot := repo.Snapshot()

			assert.EqualValues(t, 2, snapshot.Counters["PollCount"])
		})
	})

	t.Run("Тест метода Snapshot", func(t *testing.T) {
		t.Run("Должен вернуть копию snapshot", func(t *testing.T) {
			setup(t)

			repo.SaveGauge("Alloc", 10.5)
			snapshot := repo.Snapshot()
			snapshot.Gauges["Alloc"] = 99.9

			nextSnapshot := repo.Snapshot()

			assert.Equal(t, 10.5, nextSnapshot.Gauges["Alloc"])
		})
	})
}
