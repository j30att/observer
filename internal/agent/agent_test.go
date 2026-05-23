package agent_test

import (
	"context"
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
		})
	})

	t.Run("Тест метода Run", func(t *testing.T) {
		t.Run("Должен собирать и отправлять метрики пока контекст не отменён", func(t *testing.T) {
			cfg := config.NewAgentConfig()
			cfg.PollInterval = 10 * time.Millisecond
			cfg.ReportInterval = 15 * time.Millisecond
			setup(t, cfg)

			collector.EXPECT().Collect(store).Return(nil).Maybe()
			sender.EXPECT().Send(mock.Anything, mock.MatchedBy(func(agentmodel.MetricsSnapshot) bool {
				return true
			})).Return(nil).Maybe()

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
			defer cancel()

			err := app.Run(ctx)

			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.GreaterOrEqual(t, len(collector.Calls), 1)
			assert.GreaterOrEqual(t, len(sender.Calls), 1)
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
	})
}
