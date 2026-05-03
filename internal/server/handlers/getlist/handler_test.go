package getlist_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/model"
	repositorymocks "j30att/observer/internal/server/repository/mocks"
)

func TestGetListMetricHandler(t *testing.T) {
	var (
		handler *getlist.Handler
		repo    *repositorymocks.MockMetricsRepository
	)

	setup := func(t *testing.T) {
		t.Helper()

		repo = repositorymocks.NewMockMetricsRepository(t)
		handler = getlist.New(repo)
	}

	t.Run("Тест метода Execute", func(t *testing.T) {
		t.Run("Должен вернуть список метрик", func(t *testing.T) {
			setup(t)

			value := 12.5
			delta := int64(1)
			expected := []model.Metrics{
				{ID: "Alloc", MType: model.Gauge, Value: &value},
				{ID: "PollCount", MType: model.Counter, Delta: &delta},
			}
			repo.EXPECT().List().Return(expected)

			result := handler.Execute()

			assert.Equal(t, expected, result)
		})
	})
}
