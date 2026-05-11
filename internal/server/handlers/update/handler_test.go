package update_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/model"
	repositorymocks "j30att/observer/internal/server/repository/mocks"
)

func TestUpdateMetricHandler(t *testing.T) {
	var (
		handler *update.Handler
		repo    *repositorymocks.MockMetricsRepository
	)

	setup := func(t *testing.T) {
		t.Helper()

		repo = repositorymocks.NewMockMetricsRepository(t)
		handler = update.New(repo)
	}

	t.Run("Тест метода Execute", func(t *testing.T) {
		t.Run("Должен сохранить gauge", func(t *testing.T) {
			setup(t)

			repo.EXPECT().SaveGauge(context.Background(), "Alloc", 12.5).Return(nil)

			err := handler.Execute(context.Background(), model.Gauge, "Alloc", "12.5")

			require.NoError(t, err)
		})

		t.Run("Должен сохранить counter", func(t *testing.T) {
			setup(t)

			repo.EXPECT().SaveCounter(context.Background(), "PollCount", int64(2)).Return(nil)

			err := handler.Execute(context.Background(), model.Counter, "PollCount", "2")

			require.NoError(t, err)
		})

		t.Run("Должен вернуть ошибку если тип метрики не поддерживается", func(t *testing.T) {
			setup(t)

			err := handler.Execute(context.Background(), "summary", "Alloc", "12.5")

			require.ErrorIs(t, err, update.ErrUnsupportedMetricType)
			repo.AssertNotCalled(t, "SaveGauge", mock.Anything, mock.Anything)
			repo.AssertNotCalled(t, "SaveCounter", mock.Anything, mock.Anything)
		})

		t.Run("Должен вернуть ошибку если counter не число", func(t *testing.T) {
			setup(t)

			err := handler.Execute(context.Background(), model.Counter, "PollCount", "abc")

			require.Error(t, err)
			repo.AssertNotCalled(t, "SaveCounter", mock.Anything, mock.Anything)
		})

		t.Run("Должен вернуть ошибку репозитория", func(t *testing.T) {
			setup(t)

			saveErr := errors.New("save failed")
			repo.EXPECT().SaveGauge(context.Background(), "Alloc", 12.5).Return(saveErr)

			err := handler.Execute(context.Background(), model.Gauge, "Alloc", "12.5")

			assert.ErrorIs(t, err, saveErr)
		})
	})

	t.Run("Тест метода ExecuteBatch", func(t *testing.T) {
		t.Run("Должен сохранить batch", func(t *testing.T) {
			setup(t)

			value := 12.5
			delta := int64(2)
			metrics := []model.Metrics{
				{ID: "Alloc", MType: model.Gauge, Value: &value},
				{ID: "PollCount", MType: model.Counter, Delta: &delta},
			}
			repo.EXPECT().SaveBatch(context.Background(), metrics).Return(nil)

			err := handler.ExecuteBatch(context.Background(), metrics)

			require.NoError(t, err)
		})

		t.Run("Должен отклонить невалидную метрику до сохранения", func(t *testing.T) {
			setup(t)

			err := handler.ExecuteBatch(context.Background(), []model.Metrics{{ID: "Alloc", MType: model.Gauge}})

			require.ErrorIs(t, err, update.ErrUnsupportedMetricType)
			repo.AssertNotCalled(t, "SaveBatch", mock.Anything)
		})
	})
}
