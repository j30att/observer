package getlist_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/repository"
)

func TestListMetricsReturnsAllKnownMetrics(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	err = repo.SaveCounter("PollCount", 1)
	require.NoError(t, err)

	query := getlist.New(repo)
	metrics := query.Execute()
	require.Len(t, metrics, 2)
}
