package agent_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/agent"
	agentmodel "j30att/observer/internal/agent/model"
	"j30att/observer/internal/agent/repository"
	"j30att/observer/internal/config"
)

type stubCollector struct {
	mu    sync.Mutex
	calls int
}

func (c *stubCollector) Collect(store *repository.MetricsRepository) error {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()

	store.SaveGauge("Alloc", 12.5)
	store.SaveCounter(agentmodel.PollCountMetric, 1)

	return nil
}

func (c *stubCollector) CallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.calls
}

type stubSender struct {
	mu        sync.Mutex
	calls     int
	snapshots []agentmodel.MetricsSnapshot
}

func (s *stubSender) Send(_ context.Context, snapshot agentmodel.MetricsSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	s.snapshots = append(s.snapshots, snapshot)

	return nil
}

func (s *stubSender) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.calls
}

func (s *stubSender) LastSnapshot() agentmodel.MetricsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.snapshots[len(s.snapshots)-1]
}

func TestAgentPollCollectsMetricsIntoRepository(t *testing.T) {
	cfg := config.AgentConfig{}
	repo := repository.NewMetricsRepository()
	collector := &stubCollector{}
	sender := &stubSender{}
	app := agent.New(cfg, repo, collector, sender)

	err := app.Poll()
	require.NoError(t, err)

	snapshot := repo.Snapshot()
	require.Equal(t, 12.5, snapshot.Gauges["Alloc"])
	require.EqualValues(t, 1, snapshot.Counters[agentmodel.PollCountMetric])
	require.Equal(t, 1, collector.CallCount())
}

func TestAgentReportSendsRepositorySnapshot(t *testing.T) {
	cfg := config.AgentConfig{}
	repo := repository.NewMetricsRepository()
	repo.SaveGauge("Alloc", 12.5)
	repo.SaveCounter(agentmodel.PollCountMetric, 2)

	collector := &stubCollector{}
	sender := &stubSender{}
	app := agent.New(cfg, repo, collector, sender)

	err := app.Report(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, sender.CallCount())
	require.Equal(t, 12.5, sender.LastSnapshot().Gauges["Alloc"])
	require.EqualValues(t, 2, sender.LastSnapshot().Counters[agentmodel.PollCountMetric])
}

func TestAgentRunPollsAndReportsUntilContextCancelled(t *testing.T) {
	cfg := config.AgentConfig{
		PollInterval:   10 * time.Millisecond,
		ReportInterval: 15 * time.Millisecond,
	}
	repo := repository.NewMetricsRepository()
	collector := &stubCollector{}
	sender := &stubSender{}
	app := agent.New(cfg, repo, collector, sender)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	err := app.Run(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.GreaterOrEqual(t, collector.CallCount(), 1)
	require.GreaterOrEqual(t, sender.CallCount(), 1)
}
