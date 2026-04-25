package repository_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/agent/repository"
)

func TestSaveGaugeKeepsLatestValue(t *testing.T) {
	repo := repository.NewMetricsRepository()

	repo.SaveGauge("Alloc", 10.5)
	repo.SaveGauge("Alloc", 12.5)

	snapshot := repo.Snapshot()
	require.Equal(t, 12.5, snapshot.Gauges["Alloc"])
}

func TestSaveCounterAccumulatesValue(t *testing.T) {
	repo := repository.NewMetricsRepository()

	repo.SaveCounter("PollCount", 1)
	repo.SaveCounter("PollCount", 1)

	snapshot := repo.Snapshot()
	require.EqualValues(t, 2, snapshot.Counters["PollCount"])
}

func TestSnapshotReturnsCopy(t *testing.T) {
	repo := repository.NewMetricsRepository()
	repo.SaveGauge("Alloc", 10.5)

	snapshot := repo.Snapshot()
	snapshot.Gauges["Alloc"] = 99.9

	nextSnapshot := repo.Snapshot()
	require.Equal(t, 10.5, nextSnapshot.Gauges["Alloc"])
}
