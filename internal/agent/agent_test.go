package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/agent"
	agentmocks "j30att/observer/internal/agent/mocks"
	agentmodel "j30att/observer/internal/agent/model"
	"j30att/observer/internal/agent/repository"
	"j30att/observer/internal/config"
)

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
		app = agent.New(cfg, store, collector, sender)
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
			setup(t, config.AgentConfig{
				PollInterval:   10 * time.Millisecond,
				ReportInterval: 15 * time.Millisecond,
			})

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
	})
}
