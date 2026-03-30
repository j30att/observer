package agent

import (
	"context"
	"log"
	"time"

	"j30att/observer/internal/agent/model"
	"j30att/observer/internal/agent/repository"
	"j30att/observer/internal/config"
)

type Collector interface {
	Collect(store *repository.MetricsRepository) error
}

type Sender interface {
	Send(ctx context.Context, snapshot model.MetricsSnapshot) error
}

type Agent struct {
	store          *repository.MetricsRepository
	collector      Collector
	sender         Sender
	pollInterval   time.Duration
	reportInterval time.Duration
}

func New(cfg config.AgentConfig, store *repository.MetricsRepository, collector Collector, sender Sender) *Agent {
	return &Agent{
		store:          store,
		collector:      collector,
		sender:         sender,
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
	}
}

func (a *Agent) PollInterval() time.Duration {
	return a.pollInterval
}

func (a *Agent) ReportInterval() time.Duration {
	return a.reportInterval
}

func (a *Agent) Store() *repository.MetricsRepository {
	return a.store
}

func (a *Agent) Poll() error {
	return a.collector.Collect(a.store)
}

func (a *Agent) Report(ctx context.Context) error {
	return a.sender.Send(ctx, a.store.Snapshot())
}

func (a *Agent) Run(ctx context.Context) error {
	go a.runPollLoop(ctx)
	go a.runReportLoop(ctx)

	<-ctx.Done()
	return ctx.Err()
}

func (a *Agent) runPollLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := a.Poll(); err != nil {
			log.Printf("poll metrics: %v", err)
		}

		time.Sleep(a.pollInterval)
	}
}

func (a *Agent) runReportLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := a.Report(ctx); err != nil {
			log.Printf("report metrics: %v", err)
		}

		time.Sleep(a.reportInterval)
	}
}
