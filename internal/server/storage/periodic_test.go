package storage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/storage"
	storagemocks "j30att/observer/internal/server/storage/mocks"
)

func TestPeriodicSaver(t *testing.T) {
	var (
		saver  *storage.PeriodicSaver
		source *storagemocks.MockMetricsLister
		target *storagemocks.MockMetricsSaver
	)

	setup := func(t *testing.T, interval time.Duration) {
		t.Helper()

		source = storagemocks.NewMockMetricsLister(t)
		target = storagemocks.NewMockMetricsSaver(t)
		saver = storage.NewPeriodicSaver(interval, source, target, zerolog.Nop())
	}

	t.Run("Тест метода Save", func(t *testing.T) {
		t.Run("Должен сохранить текущие метрики", func(t *testing.T) {
			setup(t, time.Second)

			value := 12.5
			metrics := []model.Metrics{{ID: "Alloc", MType: model.Gauge, Value: &value}}
			source.EXPECT().List().Return(metrics)
			target.EXPECT().Save(metrics).Return(nil)

			err := saver.Save()

			require.NoError(t, err)
		})
	})

	t.Run("Тест метода Run", func(t *testing.T) {
		t.Run("Должен периодически сохранять метрики", func(t *testing.T) {
			setup(t, 5*time.Millisecond)

			saved := make(chan struct{}, 1)
			value := 12.5
			metrics := []model.Metrics{{ID: "Alloc", MType: model.Gauge, Value: &value}}
			source.EXPECT().List().Return(metrics).Maybe()
			target.EXPECT().Save(metrics).Run(func([]model.Metrics) {
				select {
				case saved <- struct{}{}:
				default:
				}
			}).Return(nil).Maybe()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go saver.Run(ctx)

			require.Eventually(t, func() bool {
				select {
				case <-saved:
					return true
				default:
					return false
				}
			}, time.Second, 10*time.Millisecond)
		})

		t.Run("Должен остановиться если контекст отменён", func(t *testing.T) {
			setup(t, 5*time.Millisecond)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			saver.Run(ctx)

			source.AssertNotCalled(t, "List")
			target.AssertNotCalled(t, "Save", mock.Anything)
		})

		t.Run("Должен ничего не делать если interval не положительный", func(t *testing.T) {
			setup(t, 0)

			saver.Run(context.Background())

			source.AssertNotCalled(t, "List")
			target.AssertNotCalled(t, "Save", mock.Anything)
		})

		t.Run("Должен залогировать ошибку сохранения", func(t *testing.T) {
			source = storagemocks.NewMockMetricsLister(t)
			target = storagemocks.NewMockMetricsSaver(t)
			var output bytes.Buffer
			logger := zerolog.New(&output).Level(zerolog.InfoLevel)
			saver = storage.NewPeriodicSaver(5*time.Millisecond, source, target, logger)

			saveErr := errors.New("save failed")
			metrics := []model.Metrics{}
			source.EXPECT().List().Return(metrics).Maybe()
			target.EXPECT().Save(metrics).Return(saveErr).Maybe()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go saver.Run(ctx)

			require.Eventually(t, func() bool {
				return output.Len() > 0
			}, time.Second, 10*time.Millisecond)

			var entry map[string]any
			require.NoError(t, json.NewDecoder(&output).Decode(&entry))
			assert.Equal(t, "error", entry["level"])
			assert.Equal(t, "failed to save metrics", entry["message"])
			assert.Equal(t, saveErr.Error(), entry["error"])
		})
	})
}
