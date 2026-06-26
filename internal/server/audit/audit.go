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

type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Observer interface {
	Notify(ctx context.Context, event Event) error
}

type Subject struct {
	observers []Observer
	logger    zerolog.Logger
}

func NewSubject(logger zerolog.Logger, observers ...Observer) *Subject {
	return &Subject{
		observers: observers,
		logger:    logger,
	}
}

func (s *Subject) Notify(ctx context.Context, event Event) {
	for _, observer := range s.observers {
		if err := observer.Notify(ctx, event); err != nil {
			s.logger.Error().Err(err).Msg("failed to send audit event")
		}
	}
}

func NewEvent(metricNames []string, ipAddress string) Event {
	return Event{
		Timestamp: time.Now().Unix(),
		Metrics:   metricNames,
		IPAddress: ipAddress,
	}
}

type FileObserver struct {
	path string
}

func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

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

type URLObserver struct {
	client *http.Client
	url    string
}

func NewURLObserver(url string) *URLObserver {
	return &URLObserver{
		client: &http.Client{Timeout: 5 * time.Second},
		url:    url,
	}
}

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
