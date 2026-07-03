package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"j30att/observer/internal/retry"

	"github.com/rs/zerolog"
)

const observerQueueSize = 100

type observerJob struct {
	ctx   context.Context
	event Event
}

// Event describes a successful metric update for audit observers.
type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Observer receives audit events.
type Observer interface {
	Notify(ctx context.Context, event Event) error
}

type closeObserver interface {
	Close() error
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type retryableHTTPClient struct {
	client      *http.Client
	retryDelays []time.Duration
}

func newRetryableHTTPClient(client *http.Client, retryDelays []time.Duration) *retryableHTTPClient {
	return &retryableHTTPClient{
		client:      client,
		retryDelays: retryDelays,
	}
}

func (c *retryableHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	attempt := 0
	err := retry.Do(req.Context(), c.retryDelays, isRetriableAuditTransportError, func() error {
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return fmt.Errorf("recreate request body: %w", err)
			}
			req.Body = body
		}
		attempt++

		var err error
		resp, err = c.client.Do(req)
		return err
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func isRetriableAuditTransportError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}

// Subject broadcasts audit events to registered observers.
type Subject struct {
	mu        sync.RWMutex
	observers []Observer
	queues    []chan observerJob
	closed    bool
	wg        sync.WaitGroup
	logger    zerolog.Logger
}

// NewSubject creates an audit event subject.
func NewSubject(logger zerolog.Logger, observers ...Observer) *Subject {
	subject := &Subject{
		observers: make([]Observer, 0, len(observers)),
		queues:    make([]chan observerJob, 0, len(observers)),
		logger:    logger,
	}

	for _, observer := range observers {
		if observer == nil {
			continue
		}

		queue := make(chan observerJob, observerQueueSize)
		subject.observers = append(subject.observers, observer)
		subject.queues = append(subject.queues, queue)
		subject.wg.Add(1)
		go subject.runObserver(observer, queue)
	}

	return subject
}

// Notify queues event delivery for all observers without waiting for delivery.
func (s *Subject) Notify(ctx context.Context, event Event) {
	job := observerJob{
		ctx:   context.WithoutCancel(ctx),
		event: event,
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		s.logger.Error().Msg("audit subject is closed; event dropped")
		return
	}

	for _, queue := range s.queues {
		select {
		case queue <- job:
		default:
			s.logger.Error().Msg("audit observer queue is full; event dropped")
		}
	}
}

func (s *Subject) runObserver(observer Observer, queue <-chan observerJob) {
	defer s.wg.Done()
	for job := range queue {
		if err := observer.Notify(job.ctx, job.event); err != nil {
			s.logger.Error().Err(err).Msg("failed to send audit event")
		}
	}
}

// Close stops audit workers after queued events are delivered and closes observers.
func (s *Subject) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	for _, queue := range s.queues {
		close(queue)
	}
	s.mu.Unlock()

	s.wg.Wait()

	var closeErr error
	for _, observer := range s.observers {
		closer, ok := observer.(closeObserver)
		if !ok {
			continue
		}

		if err := closer.Close(); err != nil {
			closeErr = err
			s.logger.Error().Err(err).Msg("failed to close audit observer")
		}
	}

	return closeErr
}

// NewEvent creates an audit event for updated metric names and client IP.
func NewEvent(metricNames []string, ipAddress string) Event {
	return Event{
		Timestamp: time.Now().Unix(),
		Metrics:   metricNames,
		IPAddress: ipAddress,
	}
}

// FileObserver appends audit events to a local JSON-lines file.
type FileObserver struct {
	mu   sync.Mutex
	path string
	file *os.File
}

// NewFileObserver creates a file audit observer.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Notify writes one audit event line to the configured file.
func (o *FileObserver) Notify(_ context.Context, event Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	file, err := o.open()
	if err != nil {
		return err
	}

	if _, err := file.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("write audit file: %w", err)
	}

	return nil
}

func (o *FileObserver) open() (*os.File, error) {
	if o.file != nil {
		return o.file, nil
	}

	file, err := os.OpenFile(o.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}

	o.file = file
	return file, nil
}

// Close closes the audit file if it has been opened.
func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.file == nil {
		return nil
	}

	if err := o.file.Close(); err != nil {
		return fmt.Errorf("close audit file: %w", err)
	}
	o.file = nil

	return nil
}

// URLObserver posts audit events to an HTTP endpoint.
type URLObserver struct {
	client httpDoer
	url    string
}

// NewURLObserver creates an HTTP audit observer.
func NewURLObserver(url string) *URLObserver {
	return &URLObserver{
		client: newRetryableHTTPClient(
			&http.Client{Timeout: 5 * time.Second},
			retry.DefaultDelays,
		),
		url: url,
	}
}

// Notify posts one audit event as JSON.
func (o *URLObserver) Notify(ctx context.Context, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("send audit request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected audit response status: %s", resp.Status)
	}

	return nil
}
