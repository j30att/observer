package repository_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

func TestSyncPersistentRepositorySavesSnapshotAfterGaugeUpdate(t *testing.T) {
	repo := repository.NewMetricsRepository()
	saver := &fakeMetricsSaver{}
	persistentRepo := repository.NewSyncPersistentRepository(repo, saver)

	require.NoError(t, persistentRepo.SaveGauge("Alloc", 12.5))

	snapshots := saver.savedSnapshots()
	require.Len(t, snapshots, 1)
	require.Len(t, snapshots[0], 1)
	require.Equal(t, "Alloc", snapshots[0][0].ID)
	require.Equal(t, model.Gauge, snapshots[0][0].MType)
	require.NotNil(t, snapshots[0][0].Value)
	require.Equal(t, 12.5, *snapshots[0][0].Value)
}

func TestSyncPersistentRepositorySavesSnapshotAfterCounterUpdate(t *testing.T) {
	repo := repository.NewMetricsRepository()
	saver := &fakeMetricsSaver{}
	persistentRepo := repository.NewSyncPersistentRepository(repo, saver)

	require.NoError(t, persistentRepo.SaveCounter("PollCount", 2))
	require.NoError(t, persistentRepo.SaveCounter("PollCount", 3))

	snapshots := saver.savedSnapshots()
	require.Len(t, snapshots, 2)
	require.Len(t, snapshots[1], 1)
	require.Equal(t, "PollCount", snapshots[1][0].ID)
	require.Equal(t, model.Counter, snapshots[1][0].MType)
	require.NotNil(t, snapshots[1][0].Delta)
	require.EqualValues(t, 5, *snapshots[1][0].Delta)
}

func TestSyncPersistentRepositoryReturnsSaveError(t *testing.T) {
	saveErr := errors.New("save failed")
	repo := repository.NewMetricsRepository()
	saver := &fakeMetricsSaver{err: saveErr}
	persistentRepo := repository.NewSyncPersistentRepository(repo, saver)

	err := persistentRepo.SaveGauge("Alloc", 12.5)

	require.ErrorIs(t, err, saveErr)
}

func TestSyncPersistentRepositoryDelegatesLoadAndList(t *testing.T) {
	repo := repository.NewMetricsRepository()
	saver := &fakeMetricsSaver{}
	persistentRepo := repository.NewSyncPersistentRepository(repo, saver)

	require.NoError(t, persistentRepo.SaveGauge("Alloc", 12.5))

	metric, err := persistentRepo.Load(model.Gauge, "Alloc")
	require.NoError(t, err)
	require.NotNil(t, metric.Value)
	require.Equal(t, 12.5, *metric.Value)

	metrics := persistentRepo.List()
	require.Len(t, metrics, 1)
	require.Equal(t, "Alloc", metrics[0].ID)
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
