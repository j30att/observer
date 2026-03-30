package senders

import (
	"context"
	"fmt"
	commonmodel "j30att/observer/internal/server/model"
	"net/http"
	"strconv"
	"strings"

	agentmodel "j30att/observer/internal/agent/model"
)

type HTTPSender struct {
	address string
	client  *http.Client
}

func NewHTTPSender(address string) *HTTPSender {
	return &HTTPSender{
		address: strings.TrimRight(address, "/"),
		client:  &http.Client{},
	}
}

func (s *HTTPSender) Send(ctx context.Context, snapshot agentmodel.MetricsSnapshot) error {
	for name, value := range snapshot.Gauges {
		rawValue := strconv.FormatFloat(value, 'f', -1, 64)
		if err := s.sendMetric(ctx, commonmodel.Gauge, name, rawValue); err != nil {
			return err
		}
	}

	for name, value := range snapshot.Counters {
		rawValue := strconv.FormatInt(value, 10)
		if err := s.sendMetric(ctx, commonmodel.Counter, name, rawValue); err != nil {
			return err
		}
	}

	return nil
}

func (s *HTTPSender) sendMetric(ctx context.Context, metricType, name, value string) error {
	url := fmt.Sprintf("%s/update/%s/%s/%s", s.address, metricType, name, value)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
