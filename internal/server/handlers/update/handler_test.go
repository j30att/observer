package update_test

import (
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

			repo.EXPECT().SaveGauge("Alloc", 12.5).Return(nil)

			err := handler.Execute(model.Gauge, "Alloc", "12.5")

			require.NoError(t, err)
		})

		t.Run("Должен сохранить counter", func(t *testing.T) {
			setup(t)

			repo.EXPECT().SaveCounter("PollCount", int64(2)).Return(nil)

			err := handler.Execute(model.Counter, "PollCount", "2")

			require.NoError(t, err)
		})

		t.Run("Должен вернуть ошибку если тип метрики не поддерживается", func(t *testing.T) {
			setup(t)

			err := handler.Execute("summary", "Alloc", "12.5")

			require.ErrorIs(t, err, update.ErrUnsupportedMetricType)
			repo.AssertNotCalled(t, "SaveGauge", mock.Anything, mock.Anything)
			repo.AssertNotCalled(t, "SaveCounter", mock.Anything, mock.Anything)
		})

		t.Run("Должен вернуть ошибку если counter не число", func(t *testing.T) {
			setup(t)

			err := handler.Execute(model.Counter, "PollCount", "abc")

			require.Error(t, err)
			repo.AssertNotCalled(t, "SaveCounter", mock.Anything, mock.Anything)
		})

		t.Run("Должен вернуть ошибку репозитория", func(t *testing.T) {
			setup(t)

			saveErr := errors.New("save failed")
			repo.EXPECT().SaveGauge("Alloc", 12.5).Return(saveErr)

			err := handler.Execute(model.Gauge, "Alloc", "12.5")

			assert.ErrorIs(t, err, saveErr)
		})
	})
}
