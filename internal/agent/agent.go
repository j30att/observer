package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"j30att/observer/internal/agent/model"
	"j30att/observer/internal/agent/repository"
	"j30att/observer/internal/config"
)

const gracefulShutdownTimeout = 30 * time.Second

var errGracefulShutdownTimeout = errors.New("agent graceful shutdown timed out")

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
	return a.sendSnapshot(ctx, a.store.TakeSnapshot())
}

// Run starts polling and reporting loops until the context is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	reports := make(chan model.MetricsSnapshot, a.rateLimit)
	sendCtx, cancelSends := context.WithCancelCause(context.WithoutCancel(ctx))
	defer cancelSends(nil)

	var workerWG sync.WaitGroup
	for range a.rateLimit {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			a.runReportWorker(sendCtx, reports)
		}()
	}

	var pollWG sync.WaitGroup
	for _, collector := range a.collectors {
		pollWG.Add(1)
		go func(collector Collector) {
			defer pollWG.Done()
			a.runPollLoop(ctx, collector)
		}(collector)
	}

	var reportWG sync.WaitGroup
	reportWG.Add(1)
	go func() {
		defer reportWG.Done()
		a.runReportLoop(ctx, reports)
	}()

	<-ctx.Done()
	shutdownTimer := time.AfterFunc(gracefulShutdownTimeout, func() {
		cancelSends(errGracefulShutdownTimeout)
	})
	defer shutdownTimer.Stop()

	if !waitGroup(sendCtx, &pollWG) || !waitGroup(sendCtx, &reportWG) {
		return context.Cause(sendCtx)
	}

	close(reports)
	if !waitGroup(sendCtx, &workerWG) {
		return context.Cause(sendCtx)
	}

	if err := a.Report(sendCtx); err != nil {
		if cause := context.Cause(sendCtx); cause != nil {
			return cause
		}
		return fmt.Errorf("send final metrics report: %w", err)
	}

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

		snapshot := a.store.TakeSnapshot()
		select {
		case reports <- snapshot:
		case <-ctx.Done():
			a.store.RestoreCounters(snapshot.Counters)
			return
		}

		if !sleepOrDone(ctx, a.reportInterval) {
			return
		}
	}
}

func (a *Agent) runReportWorker(ctx context.Context, reports <-chan model.MetricsSnapshot) {
	for snapshot := range reports {
		if err := a.sendSnapshot(ctx, snapshot); err != nil {
			a.logger.Error().Err(err).Msg("failed to report metrics")
		}
	}
}

func (a *Agent) sendSnapshot(ctx context.Context, snapshot model.MetricsSnapshot) error {
	if err := a.sender.Send(ctx, snapshot); err != nil {
		a.store.RestoreCounters(snapshot.Counters)
		return err
	}

	return nil
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

func waitGroup(ctx context.Context, wg *sync.WaitGroup) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}
