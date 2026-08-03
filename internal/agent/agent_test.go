package agent_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/agent"
	"j30att/observer/internal/agent/collectors"
	agentmocks "j30att/observer/internal/agent/mocks"
	agentmodel "j30att/observer/internal/agent/model"
	"j30att/observer/internal/agent/repository"
	"j30att/observer/internal/config"
)

type trackingSender struct {
	active    int64
	maxActive int64
	calls     int64
	delay     time.Duration
}

type collectorFunc func(*repository.MetricsRepository) error

func (f collectorFunc) Collect(store *repository.MetricsRepository) error {
	return f(store)
}

type recordingSender struct {
	mu              sync.Mutex
	snapshots       []agentmodel.MetricsSnapshot
	canceledContext bool
	err             error
}

func (s *recordingSender) Send(ctx context.Context, snapshot agentmodel.MetricsSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshots = append(s.snapshots, snapshot)
	s.canceledContext = s.canceledContext || ctx.Err() != nil
	return s.err
}

func (s *recordingSender) state() ([]agentmodel.MetricsSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]agentmodel.MetricsSnapshot(nil), s.snapshots...), s.canceledContext
}

func (s *trackingSender) Send(ctx context.Context, _ agentmodel.MetricsSnapshot) error {
	active := atomic.AddInt64(&s.active, 1)
	defer atomic.AddInt64(&s.active, -1)
	atomic.AddInt64(&s.calls, 1)

	for {
		maxActive := atomic.LoadInt64(&s.maxActive)
		if active <= maxActive || atomic.CompareAndSwapInt64(&s.maxActive, maxActive, active) {
			break
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.delay):
		return nil
	}
}

func TestAgent(t *testing.T) {
	var (
		app       *agent.Agent
		store     *repository.MetricsRepository
		collector *agentmocks.MockCollector
		sender    *agentmocks.MockSender
	)

	setup := func(t *testing.T, cfg config.AgentConfig) {
		t.Helper()

		store = repository.NewMetricsRepository()
		collector = agentmocks.NewMockCollector(t)
		sender = agentmocks.NewMockSender(t)
		app = agent.New(cfg, store, []agent.Collector{collector}, sender, zerolog.Nop())
	}

	t.Run("Тест метода Poll", func(t *testing.T) {
		t.Run("Должен собрать метрики в repository", func(t *testing.T) {
			setup(t, config.AgentConfig{})

			collector.EXPECT().Collect(store).Run(func(store *repository.MetricsRepository) {
				store.SaveGauge("Alloc", 12.5)
				store.SaveCounter(agentmodel.PollCountMetric, 1)
			}).Return(nil)

			err := app.Poll()

			require.NoError(t, err)
			snapshot := store.Snapshot()
			assert.Equal(t, 12.5, snapshot.Gauges["Alloc"])
			assert.EqualValues(t, 1, snapshot.Counters[agentmodel.PollCountMetric])
		})
	})

	t.Run("Тест метода Report", func(t *testing.T) {
		t.Run("Должен отправить snapshot repository", func(t *testing.T) {
			setup(t, config.AgentConfig{})

			store.SaveGauge("Alloc", 12.5)
			store.SaveCounter(agentmodel.PollCountMetric, 2)
			sender.EXPECT().Send(mock.Anything, mock.MatchedBy(func(snapshot agentmodel.MetricsSnapshot) bool {
				return snapshot.Gauges["Alloc"] == 12.5 &&
					snapshot.Counters[agentmodel.PollCountMetric] == 2
			})).Return(nil)

			err := app.Report(context.Background())

			require.NoError(t, err)
			assert.Empty(t, store.Snapshot().Counters)
		})

		t.Run("Должен отправлять только новую counter delta", func(t *testing.T) {
			store := repository.NewMetricsRepository()
			sender := &recordingSender{}
			app := agent.New(config.AgentConfig{}, store, nil, sender, zerolog.Nop())
			store.SaveCounter(agentmodel.PollCountMetric, 2)

			require.NoError(t, app.Report(context.Background()))
			store.SaveCounter(agentmodel.PollCountMetric, 1)
			require.NoError(t, app.Report(context.Background()))

			snapshots, _ := sender.state()
			require.Len(t, snapshots, 2)
			assert.EqualValues(t, 2, snapshots[0].Counters[agentmodel.PollCountMetric])
			assert.EqualValues(t, 1, snapshots[1].Counters[agentmodel.PollCountMetric])
		})

		t.Run("Должен вернуть counter delta после ошибки", func(t *testing.T) {
			sendErr := errors.New("send failed")
			store := repository.NewMetricsRepository()
			sender := &recordingSender{err: sendErr}
			app := agent.New(config.AgentConfig{}, store, nil, sender, zerolog.Nop())
			store.SaveCounter(agentmodel.PollCountMetric, 2)

			err := app.Report(context.Background())

			require.ErrorIs(t, err, sendErr)
			assert.EqualValues(t, 2, store.Snapshot().Counters[agentmodel.PollCountMetric])
		})
	})

	t.Run("Тест метода Run", func(t *testing.T) {
		t.Run("Должен собирать и отправлять метрики пока контекст не отменён", func(t *testing.T) {
			cfg := config.NewAgentConfig()
			cfg.PollInterval = 10 * time.Millisecond
			cfg.ReportInterval = 15 * time.Millisecond
			store := repository.NewMetricsRepository()
			var collectCalls atomic.Int64
			collector := collectorFunc(func(*repository.MetricsRepository) error {
				collectCalls.Add(1)
				return nil
			})
			sender := &recordingSender{}
			app := agent.New(cfg, store, []agent.Collector{collector}, sender, zerolog.Nop())

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
			defer cancel()

			err := app.Run(ctx)

			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.GreaterOrEqual(t, collectCalls.Load(), int64(1))
			snapshots, _ := sender.state()
			assert.GreaterOrEqual(t, len(snapshots), 1)
		})

		t.Run("Должен ограничивать количество одновременных отправок", func(t *testing.T) {
			store := repository.NewMetricsRepository()
			tracker := &trackingSender{delay: 30 * time.Millisecond}
			app := agent.New(config.AgentConfig{
				PollInterval:   time.Hour,
				ReportInterval: time.Millisecond,
				RateLimit:      2,
			}, store, []agent.Collector{collectors.NoopCollector{}}, tracker, zerolog.Nop())

			ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
			defer cancel()

			err := app.Run(ctx)

			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.LessOrEqual(t, atomic.LoadInt64(&tracker.maxActive), int64(2))
			assert.GreaterOrEqual(t, atomic.LoadInt64(&tracker.calls), int64(2))
		})

		t.Run("Должен завершить сбор и отправить финальный snapshot после отмены", func(t *testing.T) {
			store := repository.NewMetricsRepository()
			started := make(chan struct{})
			release := make(chan struct{})
			collector := collectorFunc(func(store *repository.MetricsRepository) error {
				close(started)
				<-release
				store.SaveGauge("CollectedDuringShutdown", 42)
				return nil
			})
			sender := &recordingSender{}
			app := agent.New(config.AgentConfig{
				PollInterval:   time.Hour,
				ReportInterval: time.Hour,
				RateLimit:      1,
			}, store, []agent.Collector{collector}, sender, zerolog.Nop())

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() {
				done <- app.Run(ctx)
			}()

			<-started
			cancel()
			close(release)

			require.ErrorIs(t, <-done, context.Canceled)
			snapshots, canceledContext := sender.state()
			require.NotEmpty(t, snapshots)
			assert.Equal(t, float64(42), snapshots[len(snapshots)-1].Gauges["CollectedDuringShutdown"])
			assert.False(t, canceledContext)
		})

		t.Run("Должен вернуть ошибку финальной отправки", func(t *testing.T) {
			sendErr := errors.New("send failed")
			sender := &recordingSender{err: sendErr}
			app := agent.New(config.AgentConfig{
				PollInterval:   time.Hour,
				ReportInterval: time.Hour,
				RateLimit:      1,
			}, repository.NewMetricsRepository(), nil, sender, zerolog.Nop())

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := app.Run(ctx)

			require.ErrorIs(t, err, sendErr)
			assert.ErrorContains(t, err, "send final metrics report")
		})
	})
}
