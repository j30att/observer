package storage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/storage"
)

func TestPeriodicSaverSaveWritesCurrentMetrics(t *testing.T) {
	value := 12.5
	source := &fakeMetricsLister{
		metrics: []model.Metrics{
			{ID: "Alloc", MType: model.Gauge, Value: &value},
		},
	}
	target := &fakeMetricsSaver{}
	saver := storage.NewPeriodicSaver(time.Second, source, target, zerolog.Nop())

	require.NoError(t, saver.Save())

	require.Equal(t, [][]model.Metrics{
		source.metrics,
	}, target.savedSnapshots())
}

func TestPeriodicSaverRunSavesMetricsPeriodically(t *testing.T) {
	value := 12.5
	source := &fakeMetricsLister{
		metrics: []model.Metrics{
			{ID: "Alloc", MType: model.Gauge, Value: &value},
		},
	}
	target := &fakeMetricsSaver{}
	saver := storage.NewPeriodicSaver(5*time.Millisecond, source, target, zerolog.Nop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go saver.Run(ctx)

	require.Eventually(t, func() bool {
		return len(target.savedSnapshots()) > 0
	}, time.Second, 10*time.Millisecond)
}

func TestPeriodicSaverRunStopsWhenContextIsCanceled(t *testing.T) {
	source := &fakeMetricsLister{}
	target := &fakeMetricsSaver{}
	saver := storage.NewPeriodicSaver(5*time.Millisecond, source, target, zerolog.Nop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	saver.Run(ctx)

	require.Empty(t, target.savedSnapshots())
}

func TestPeriodicSaverRunDoesNothingWhenIntervalIsNotPositive(t *testing.T) {
	source := &fakeMetricsLister{}
	target := &fakeMetricsSaver{}
	saver := storage.NewPeriodicSaver(0, source, target, zerolog.Nop())

	saver.Run(context.Background())

	require.Empty(t, target.savedSnapshots())
}

func TestPeriodicSaverRunLogsErrorWhenSaveFails(t *testing.T) {
	saveErr := errors.New("save failed")
	source := &fakeMetricsLister{}
	target := &fakeMetricsSaver{err: saveErr}
	var output bytes.Buffer
	logger := zerolog.New(&output).Level(zerolog.InfoLevel)
	saver := storage.NewPeriodicSaver(5*time.Millisecond, source, target, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go saver.Run(ctx)

	require.Eventually(t, func() bool {
		return output.Len() > 0
	}, time.Second, 10*time.Millisecond)

	var entry map[string]any
	require.NoError(t, json.NewDecoder(&output).Decode(&entry))
	require.Equal(t, "error", entry["level"])
	require.Equal(t, "failed to save metrics", entry["message"])
	require.Equal(t, saveErr.Error(), entry["error"])
}

type fakeMetricsLister struct {
	metrics []model.Metrics
}

func (l *fakeMetricsLister) List() []model.Metrics {
	return l.metrics
}

type fakeMetricsSaver struct {
	mu        sync.Mutex
	snapshots [][]model.Metrics
	err       error
}

func (s *fakeMetricsSaver) Save(metrics []model.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}

	s.snapshots = append(s.snapshots, metrics)
	return nil
}

func (s *fakeMetricsSaver) savedSnapshots() [][]model.Metrics {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([][]model.Metrics(nil), s.snapshots...)
}
