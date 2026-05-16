package agent

import (
	"context"
	"log"
	"sync"
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
	collectors     []Collector
	sender         Sender
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
}

func New(cfg config.AgentConfig, store *repository.MetricsRepository, collectors []Collector, sender Sender) *Agent {
	activeCollectors := make([]Collector, 0, len(collectors))
	for _, collector := range collectors {
		if collector != nil {
			activeCollectors = append(activeCollectors, collector)
		}
	}

	rateLimit := cfg.RateLimit
	if rateLimit <= 0 {
		rateLimit = 1
	}

	return &Agent{
		store:          store,
		collectors:     activeCollectors,
		sender:         sender,
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
		rateLimit:      rateLimit,
	}
}

func (a *Agent) PollInterval() time.Duration {
	return a.pollInterval
}

func (a *Agent) ReportInterval() time.Duration {
	return a.reportInterval
}

func (a *Agent) RateLimit() int {
	return a.rateLimit
}

func (a *Agent) Store() *repository.MetricsRepository {
	return a.store
}

func (a *Agent) Poll() error {
	for _, collector := range a.collectors {
		if err := collector.Collect(a.store); err != nil {
			return err
		}
	}

	return nil
}

func (a *Agent) Report(ctx context.Context) error {
	return a.sender.Send(ctx, a.store.Snapshot())
}

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

	for workerID := 0; workerID < a.rateLimit; workerID++ {
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
			log.Printf("poll metrics: %v", err)
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
				log.Printf("report metrics: %v", err)
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
