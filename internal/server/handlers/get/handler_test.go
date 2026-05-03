package get_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
	repositorymocks "j30att/observer/internal/server/repository/mocks"
)

func TestGetMetricHandler(t *testing.T) {
	var (
		handler *get.Handler
		repo    *repositorymocks.MockMetricsRepository
	)

	setup := func(t *testing.T) {
		t.Helper()

		repo = repositorymocks.NewMockMetricsRepository(t)
		handler = get.New(repo)
	}

	t.Run("Тест метода Execute", func(t *testing.T) {
		t.Run("Должен вернуть метрику", func(t *testing.T) {
			setup(t)

			value := 12.5
			expected := model.Metrics{ID: "Alloc", MType: model.Gauge, Value: &value}
			repo.EXPECT().Load(model.Gauge, "Alloc").Return(expected, nil)

			result, err := handler.Execute(model.Gauge, "Alloc")

			require.NoError(t, err)
			assert.Equal(t, expected, result)
		})

		t.Run("Должен вернуть ошибку not found уровня handler", func(t *testing.T) {
			setup(t)

			repo.EXPECT().Load(model.Gauge, "Alloc").Return(model.Metrics{}, repository.ErrMetricNotFound)

			_, err := handler.Execute(model.Gauge, "Alloc")

			require.ErrorIs(t, err, get.ErrMetricNotFound)
		})

		t.Run("Должен вернуть ошибку репозитория", func(t *testing.T) {
			setup(t)

			loadErr := errors.New("database unavailable")
			repo.EXPECT().Load(model.Gauge, "Alloc").Return(model.Metrics{}, loadErr)

			_, err := handler.Execute(model.Gauge, "Alloc")

			require.ErrorIs(t, err, loadErr)
		})
	})
}
