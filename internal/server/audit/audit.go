package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
)

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

// Subject broadcasts audit events to registered observers.
type Subject struct {
	observers []Observer
	logger    zerolog.Logger
}

// NewSubject creates an audit event subject.
func NewSubject(logger zerolog.Logger, observers ...Observer) *Subject {
	return &Subject{
		observers: observers,
		logger:    logger,
	}
}

// Notify sends event to all observers and logs observer failures.
func (s *Subject) Notify(ctx context.Context, event Event) {
	for _, observer := range s.observers {
		if err := observer.Notify(ctx, event); err != nil {
			s.logger.Error().Err(err).Msg("failed to send audit event")
		}
	}
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
	path string
}

// NewFileObserver creates a file audit observer.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Notify writes one audit event line to the configured file.
func (o *FileObserver) Notify(_ context.Context, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	file, err := os.OpenFile(o.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open audit file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("write audit file: %w", err)
	}

	return nil
}

// URLObserver posts audit events to an HTTP endpoint.
type URLObserver struct {
	client *http.Client
	url    string
}

// NewURLObserver creates an HTTP audit observer.
func NewURLObserver(url string) *URLObserver {
	return &URLObserver{
		client: &http.Client{Timeout: 5 * time.Second},
		url:    url,
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
