package repository_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
	repositorymocks "j30att/observer/internal/server/repository/mocks"
)

func TestSyncPersistentRepository(t *testing.T) {
	var (
		persistentRepo *repository.SyncPersistentRepository
		repo           *repositorymocks.MockMetricsRepository
		saver          *repositorymocks.MockMetricsSaver
	)

	setup := func(t *testing.T) {
		t.Helper()

		repo = repositorymocks.NewMockMetricsRepository(t)
		saver = repositorymocks.NewMockMetricsSaver(t)
		persistentRepo = repository.NewSyncPersistentRepository(repo, saver)
	}

	t.Run("Тест метода SaveGauge", func(t *testing.T) {
		t.Run("Должен сохранить gauge и snapshot", func(t *testing.T) {
			setup(t)

			value := 12.5
			snapshot := []model.Metrics{{ID: "Alloc", MType: model.Gauge, Value: &value}}
			repo.EXPECT().SaveGauge("Alloc", value).Return(nil)
			repo.EXPECT().List().Return(snapshot)
			saver.EXPECT().Save(snapshot).Return(nil)

			err := persistentRepo.SaveGauge("Alloc", value)

			require.NoError(t, err)
		})

		t.Run("Должен вернуть ошибку репозитория", func(t *testing.T) {
			setup(t)

			saveErr := errors.New("save gauge failed")
			repo.EXPECT().SaveGauge("Alloc", 12.5).Return(saveErr)

			err := persistentRepo.SaveGauge("Alloc", 12.5)

			require.ErrorIs(t, err, saveErr)
			repo.AssertNotCalled(t, "List")
			saver.AssertNotCalled(t, "Save", mock.Anything)
		})

		t.Run("Должен вернуть ошибку сохранения snapshot", func(t *testing.T) {
			setup(t)

			value := 12.5
			saveErr := errors.New("save snapshot failed")
			snapshot := []model.Metrics{{ID: "Alloc", MType: model.Gauge, Value: &value}}
			repo.EXPECT().SaveGauge("Alloc", value).Return(nil)
			repo.EXPECT().List().Return(snapshot)
			saver.EXPECT().Save(snapshot).Return(saveErr)

			err := persistentRepo.SaveGauge("Alloc", value)

			require.ErrorIs(t, err, saveErr)
		})
	})

	t.Run("Тест метода SaveCounter", func(t *testing.T) {
		t.Run("Должен сохранить counter и snapshot", func(t *testing.T) {
			setup(t)

			delta := int64(5)
			snapshot := []model.Metrics{{ID: "PollCount", MType: model.Counter, Delta: &delta}}
			repo.EXPECT().SaveCounter("PollCount", int64(2)).Return(nil)
			repo.EXPECT().List().Return(snapshot)
			saver.EXPECT().Save(snapshot).Return(nil)

			err := persistentRepo.SaveCounter("PollCount", 2)

			require.NoError(t, err)
		})
	})

	t.Run("Тест метода SaveBatch", func(t *testing.T) {
		t.Run("Должен сохранить batch и snapshot один раз", func(t *testing.T) {
			setup(t)

			value := 12.5
			delta := int64(2)
			batch := []model.Metrics{
				{ID: "Alloc", MType: model.Gauge, Value: &value},
				{ID: "PollCount", MType: model.Counter, Delta: &delta},
			}
			repo.EXPECT().SaveBatch(batch).Return(nil)
			repo.EXPECT().List().Return(batch)
			saver.EXPECT().Save(batch).Return(nil)

			err := persistentRepo.SaveBatch(batch)

			require.NoError(t, err)
		})
	})

	t.Run("Тест делегирования", func(t *testing.T) {
		t.Run("Должен делегировать Load", func(t *testing.T) {
			setup(t)

			value := 12.5
			expected := model.Metrics{ID: "Alloc", MType: model.Gauge, Value: &value}
			repo.EXPECT().Load(model.Gauge, "Alloc").Return(expected, nil)

			result, err := persistentRepo.Load(model.Gauge, "Alloc")

			require.NoError(t, err)
			assert.Equal(t, expected, result)
		})

		t.Run("Должен делегировать List", func(t *testing.T) {
			setup(t)

			value := 12.5
			expected := []model.Metrics{{ID: "Alloc", MType: model.Gauge, Value: &value}}
			repo.EXPECT().List().Return(expected)

			result := persistentRepo.List()

			assert.Equal(t, expected, result)
		})
	})
}
