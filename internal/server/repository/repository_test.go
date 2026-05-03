package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

func TestInMemoryMetricsRepository(t *testing.T) {
	var repo *repository.InMemoryMetricsRepository

	setup := func(t *testing.T) {
		t.Helper()

		repo = repository.NewMetricsRepository()
	}

	t.Run("Тест сохранения и загрузки", func(t *testing.T) {
		t.Run("Должен сохранить и загрузить gauge", func(t *testing.T) {
			setup(t)

			err := repo.SaveGauge("Alloc", 12.5)
			require.NoError(t, err)

			metric, err := repo.Load(model.Gauge, "Alloc")

			require.NoError(t, err)
			assert.Equal(t, "Alloc", metric.ID)
			assert.Equal(t, model.Gauge, metric.MType)
			require.NotNil(t, metric.Value)
			assert.Equal(t, 12.5, *metric.Value)
		})

		t.Run("Должен накапливать counter", func(t *testing.T) {
			setup(t)

			require.NoError(t, repo.SaveCounter("PollCount", 2))
			require.NoError(t, repo.SaveCounter("PollCount", 3))

			metric, err := repo.Load(model.Counter, "PollCount")

			require.NoError(t, err)
			require.NotNil(t, metric.Delta)
			assert.EqualValues(t, 5, *metric.Delta)
		})

		t.Run("Должен вернуть ErrMetricNotFound для отсутствующей gauge", func(t *testing.T) {
			setup(t)

			_, err := repo.Load(model.Gauge, "UnknownMetric")

			require.ErrorIs(t, err, repository.ErrMetricNotFound)
		})

		t.Run("Должен вернуть ErrMetricNotFound для неподдержанного типа", func(t *testing.T) {
			setup(t)

			_, err := repo.Load("summary", "Alloc")

			require.ErrorIs(t, err, repository.ErrMetricNotFound)
		})
	})

	t.Run("Тест метода List", func(t *testing.T) {
		t.Run("Должен вернуть все сохранённые метрики", func(t *testing.T) {
			setup(t)

			require.NoError(t, repo.SaveGauge("Alloc", 12.5))
			require.NoError(t, repo.SaveCounter("PollCount", 3))

			metrics := repo.List()

			require.Len(t, metrics, 2)
		})
	})

	t.Run("Тест метода SaveBatch", func(t *testing.T) {
		t.Run("Должен сохранить batch метрик", func(t *testing.T) {
			setup(t)

			value := 12.5
			delta := int64(2)
			err := repo.SaveBatch([]model.Metrics{
				{ID: "Alloc", MType: model.Gauge, Value: &value},
				{ID: "PollCount", MType: model.Counter, Delta: &delta},
			})

			require.NoError(t, err)

			gauge, err := repo.Load(model.Gauge, "Alloc")
			require.NoError(t, err)
			require.NotNil(t, gauge.Value)
			assert.Equal(t, value, *gauge.Value)

			counter, err := repo.Load(model.Counter, "PollCount")
			require.NoError(t, err)
			require.NotNil(t, counter.Delta)
			assert.Equal(t, delta, *counter.Delta)
		})

		t.Run("Должен сохранить текущие метрики если batch невалидный", func(t *testing.T) {
			setup(t)

			require.NoError(t, repo.SaveGauge("Alloc", 12.5))
			value := 99.9

			err := repo.SaveBatch([]model.Metrics{
				{ID: "HeapAlloc", MType: model.Gauge, Value: &value},
				{ID: "PollCount", MType: model.Counter},
			})

			require.ErrorIs(t, err, repository.ErrInvalidMetric)

			_, err = repo.Load(model.Gauge, "HeapAlloc")
			require.ErrorIs(t, err, repository.ErrMetricNotFound)

			metric, err := repo.Load(model.Gauge, "Alloc")
			require.NoError(t, err)
			require.NotNil(t, metric.Value)
			assert.Equal(t, 12.5, *metric.Value)
		})
	})

	t.Run("Тест метода Restore", func(t *testing.T) {
		t.Run("Должен загрузить snapshot метрик", func(t *testing.T) {
			setup(t)

			value := 12.5
			delta := int64(42)
			err := repo.Restore([]model.Metrics{
				{ID: "Alloc", MType: model.Gauge, Value: &value},
				{ID: "PollCount", MType: model.Counter, Delta: &delta},
			})

			require.NoError(t, err)

			gauge, err := repo.Load(model.Gauge, "Alloc")
			require.NoError(t, err)
			require.NotNil(t, gauge.Value)
			assert.Equal(t, value, *gauge.Value)

			counter, err := repo.Load(model.Counter, "PollCount")
			require.NoError(t, err)
			require.NotNil(t, counter.Delta)
			assert.Equal(t, delta, *counter.Delta)
		})

		t.Run("Должен заменить текущие метрики", func(t *testing.T) {
			setup(t)

			require.NoError(t, repo.SaveGauge("Alloc", 12.5))
			require.NoError(t, repo.SaveCounter("PollCount", 42))

			value := 7.5
			err := repo.Restore([]model.Metrics{{ID: "HeapAlloc", MType: model.Gauge, Value: &value}})
			require.NoError(t, err)

			_, err = repo.Load(model.Gauge, "Alloc")
			require.ErrorIs(t, err, repository.ErrMetricNotFound)
			_, err = repo.Load(model.Counter, "PollCount")
			require.ErrorIs(t, err, repository.ErrMetricNotFound)

			metric, err := repo.Load(model.Gauge, "HeapAlloc")
			require.NoError(t, err)
			require.NotNil(t, metric.Value)
			assert.Equal(t, value, *metric.Value)
		})

		t.Run("Должен вернуть ошибку для невалидной метрики", func(t *testing.T) {
			setup(t)

			err := repo.Restore([]model.Metrics{{ID: "Alloc", MType: model.Gauge}})

			require.ErrorIs(t, err, repository.ErrInvalidMetric)
		})

		t.Run("Должен сохранить текущие метрики если snapshot невалидный", func(t *testing.T) {
			setup(t)

			require.NoError(t, repo.SaveGauge("Alloc", 12.5))

			err := repo.Restore([]model.Metrics{{ID: "Alloc", MType: "summary"}})
			require.ErrorIs(t, err, repository.ErrInvalidMetric)

			metric, err := repo.Load(model.Gauge, "Alloc")
			require.NoError(t, err)
			require.NotNil(t, metric.Value)
			assert.Equal(t, 12.5, *metric.Value)
		})
	})
}
