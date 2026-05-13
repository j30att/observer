package storage_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/storage"
)

var testLogger = zerolog.Nop()

func TestFileStorage(t *testing.T) {
	var (
		path        string
		fileStorage *storage.FileStorage
	)

	setup := func(t *testing.T) {
		t.Helper()

		path = filepath.Join(t.TempDir(), "metrics.json")
		fileStorage = storage.NewFileStorage(path, testLogger)
	}

	t.Run("Тест метода Save", func(t *testing.T) {
		t.Run("Должен сохранить и загрузить метрики", func(t *testing.T) {
			setup(t)

			value := 12.5
			delta := int64(42)
			metrics := []model.Metrics{
				{ID: "Alloc", MType: model.Gauge, Value: &value},
				{ID: "PollCount", MType: model.Counter, Delta: &delta},
			}

			require.NoError(t, fileStorage.Save(metrics))

			_, loadedMetrics, err := storage.NewRestoredFileStorage(path, testLogger)

			require.NoError(t, err)
			require.Len(t, loadedMetrics, 2)
			assert.Equal(t, "Alloc", loadedMetrics[0].ID)
			assert.Equal(t, model.Gauge, loadedMetrics[0].MType)
			require.NotNil(t, loadedMetrics[0].Value)
			assert.Equal(t, value, *loadedMetrics[0].Value)
			assert.Equal(t, "PollCount", loadedMetrics[1].ID)
			assert.Equal(t, model.Counter, loadedMetrics[1].MType)
			require.NotNil(t, loadedMetrics[1].Delta)
			assert.Equal(t, delta, *loadedMetrics[1].Delta)
		})

		t.Run("Должен сохранить метрики как JSON array", func(t *testing.T) {
			setup(t)

			value := 12.5
			delta := int64(42)
			metrics := []model.Metrics{
				{ID: "LastGC", MType: model.Gauge, Value: &value},
				{ID: "NumGC", MType: model.Counter, Delta: &delta},
			}

			require.NoError(t, fileStorage.Save(metrics))

			rawBody, err := os.ReadFile(path)
			require.NoError(t, err)

			var savedMetrics []map[string]any
			require.NoError(t, json.Unmarshal(rawBody, &savedMetrics))
			assert.Equal(t, []map[string]any{
				{"id": "LastGC", "type": model.Gauge, "value": value},
				{"id": "NumGC", "type": model.Counter, "delta": float64(delta)},
			}, savedMetrics)
		})

		t.Run("Должен создать parent directories", func(t *testing.T) {
			path = filepath.Join(t.TempDir(), "nested", "dir", "metrics.json")
			fileStorage = storage.NewFileStorage(path, testLogger)

			require.NoError(t, fileStorage.Save(nil))

			assert.FileExists(t, path)
		})
	})

	t.Run("Тест восстановления", func(t *testing.T) {
		t.Run("Должен вернуть пустые метрики если файл не существует", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "missing.json")

			_, metrics, err := storage.NewRestoredFileStorage(path, testLogger)

			require.NoError(t, err)
			assert.Empty(t, metrics)
		})

		t.Run("Должен вернуть ошибку если JSON невалидный", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "metrics.json")
			require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

			_, _, err := storage.NewRestoredFileStorage(path, testLogger)

			require.Error(t, err)
		})
	})
}
