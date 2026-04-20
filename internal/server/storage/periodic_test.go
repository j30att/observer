package storage_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

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
	saver := storage.NewPeriodicSaver(time.Second, source, target, nil)

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
	saver := storage.NewPeriodicSaver(5*time.Millisecond, source, target, nil)

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
	saver := storage.NewPeriodicSaver(5*time.Millisecond, source, target, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	saver.Run(ctx)

	require.Empty(t, target.savedSnapshots())
}

func TestPeriodicSaverRunDoesNothingWhenIntervalIsNotPositive(t *testing.T) {
	source := &fakeMetricsLister{}
	target := &fakeMetricsSaver{}
	saver := storage.NewPeriodicSaver(0, source, target, nil)

	saver.Run(context.Background())

	require.Empty(t, target.savedSnapshots())
}

func TestPeriodicSaverRunCallsErrorHandlerWhenSaveFails(t *testing.T) {
	saveErr := errors.New("save failed")
	source := &fakeMetricsLister{}
	target := &fakeMetricsSaver{err: saveErr}
	errCh := make(chan error, 1)
	saver := storage.NewPeriodicSaver(5*time.Millisecond, source, target, func(err error) {
		errCh <- err
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go saver.Run(ctx)

	require.Eventually(t, func() bool {
		select {
		case err := <-errCh:
			return errors.Is(err, saveErr)
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
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
