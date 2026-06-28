package agent

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"j30att/observer/internal/agent/model"
	"j30att/observer/internal/agent/repository"
	"j30att/observer/internal/config"
)

// Collector gathers metrics and stores them in the agent repository.
type Collector interface {
	Collect(store *repository.MetricsRepository) error
}

// Sender delivers a collected metrics snapshot to an external destination.
type Sender interface {
	Send(ctx context.Context, snapshot model.MetricsSnapshot) error
}

// Agent periodically collects runtime metrics and reports them to the server.
type Agent struct {
	store          *repository.MetricsRepository
	collectors     []Collector
	sender         Sender
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
	logger         zerolog.Logger
}

// New creates an agent with the provided storage, collectors, sender, and logger.
func New(cfg config.AgentConfig, store *repository.MetricsRepository, collectors []Collector, sender Sender, logger zerolog.Logger) *Agent {
	activeCollectors := make([]Collector, 0, len(collectors))
	for _, collector := range collectors {
		if collector != nil {
			activeCollectors = append(activeCollectors, collector)
		}
	}

	return &Agent{
		store:          store,
		collectors:     activeCollectors,
		sender:         sender,
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
		rateLimit:      cfg.RateLimit,
		logger:         logger,
	}
}

// PollInterval returns the configured metrics collection interval.
func (a *Agent) PollInterval() time.Duration {
	return a.pollInterval
}

// ReportInterval returns the configured metrics reporting interval.
func (a *Agent) ReportInterval() time.Duration {
	return a.reportInterval
}

// RateLimit returns the maximum number of concurrent report requests.
func (a *Agent) RateLimit() int {
	return a.rateLimit
}

// Store returns the metrics repository used by the agent.
func (a *Agent) Store() *repository.MetricsRepository {
	return a.store
}

// Poll runs each configured collector once.
func (a *Agent) Poll() error {
	for _, collector := range a.collectors {
		if err := collector.Collect(a.store); err != nil {
			return err
		}
	}

	return nil
}

// Report sends the current metrics snapshot once.
func (a *Agent) Report(ctx context.Context) error {
	return a.sender.Send(ctx, a.store.Snapshot())
}

// Run starts polling and reporting loops until the context is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	reports := make(chan model.MetricsSnapshot, a.rateLimit)

	for _, collector := range a.collectors {
		wg.Add(1)
		go func(collector Collector) {
			defer wg.Done()
			a.runPollLoop(ctx, collector)
		}(collector)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(reports)
		a.runReportLoop(ctx, reports)
	}()

	for range a.rateLimit {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runReportWorker(ctx, reports)
		}()
	}

	<-ctx.Done()
	wg.Wait()
	return ctx.Err()
}

func (a *Agent) runPollLoop(ctx context.Context, collector Collector) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := collector.Collect(a.store); err != nil {
			a.logger.Error().Err(err).Msg("failed to poll metrics")
		}

		if !sleepOrDone(ctx, a.pollInterval) {
			return
		}
	}
}

func (a *Agent) runReportLoop(ctx context.Context, reports chan<- model.MetricsSnapshot) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case reports <- a.store.Snapshot():
		case <-ctx.Done():
			return
		}

		if !sleepOrDone(ctx, a.reportInterval) {
			return
		}
	}
}

func (a *Agent) runReportWorker(ctx context.Context, reports <-chan model.MetricsSnapshot) {
	for {
		select {
		case <-ctx.Done():
			return
		case snapshot, ok := <-reports:
			if !ok {
				return
			}
			if err := a.sender.Send(ctx, snapshot); err != nil {
				a.logger.Error().Err(err).Msg("failed to report metrics")
			}
		}
	}
}

func sleepOrDone(ctx context.Context, duration time.Duration) bool {
	if duration <= 0 {
		<-ctx.Done()
		return false
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
