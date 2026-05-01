package storage_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/storage"
)

var testLogger = zerolog.Nop()

func TestFileStorageSavesAndLoadsMetrics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	fileStorage := storage.NewFileStorage(path, testLogger)

	value := 12.5
	delta := int64(42)
	metrics := []model.Metrics{
		{
			ID:    "Alloc",
			MType: model.Gauge,
			Value: &value,
		},
		{
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &delta,
		},
	}

	require.NoError(t, fileStorage.Save(metrics))

	_, loadedMetrics, err := storage.NewRestoredFileStorage(path, testLogger)
	require.NoError(t, err)
	require.Len(t, loadedMetrics, 2)

	require.Equal(t, "Alloc", loadedMetrics[0].ID)
	require.Equal(t, model.Gauge, loadedMetrics[0].MType)
	require.NotNil(t, loadedMetrics[0].Value)
	require.Equal(t, value, *loadedMetrics[0].Value)

	require.Equal(t, "PollCount", loadedMetrics[1].ID)
	require.Equal(t, model.Counter, loadedMetrics[1].MType)
	require.NotNil(t, loadedMetrics[1].Delta)
	require.Equal(t, delta, *loadedMetrics[1].Delta)
}

func TestFileStorageSavesMetricsAsJSONArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	fileStorage := storage.NewFileStorage(path, testLogger)

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
	require.Equal(t, []map[string]any{
		{
			"id":    "LastGC",
			"type":  model.Gauge,
			"value": value,
		},
		{
			"id":    "NumGC",
			"type":  model.Counter,
			"delta": float64(delta),
		},
	}, savedMetrics)
}

func TestFileStorageCreatesParentDirectoriesOnSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "metrics.json")
	fileStorage := storage.NewFileStorage(path, testLogger)

	require.NoError(t, fileStorage.Save(nil))
	require.FileExists(t, path)
}

func TestNewRestoredFileStorageReturnsEmptyMetricsWhenFileDoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	_, metrics, err := storage.NewRestoredFileStorage(path, testLogger)

	require.NoError(t, err)
	require.Empty(t, metrics)
}

func TestNewRestoredFileStorageReturnsErrorForInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	_, _, err := storage.NewRestoredFileStorage(path, testLogger)

	require.Error(t, err)
}
