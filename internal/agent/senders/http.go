package senders

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	agentmodel "j30att/observer/internal/agent/model"
)

type HTTPSender struct {
	baseURL *url.URL
	client  *http.Client
}

func NewHTTPSender(address string) *HTTPSender {
	return &HTTPSender{
		baseURL: parseBaseURL(address),
		client:  &http.Client{},
	}
}

func parseBaseURL(address string) *url.URL {
	if !strings.Contains(address, "://") {
		return &url.URL{
			Scheme: "http",
			Host:   strings.TrimRight(address, "/"),
		}
	}

	baseURL, err := url.Parse(address)
	if err != nil {
		return &url.URL{
			Scheme: "http",
			Host:   strings.TrimRight(address, "/"),
		}
	}

	baseURL.Path = strings.TrimRight(baseURL.Path, "/")

	return baseURL
}

func (s *HTTPSender) Send(ctx context.Context, snapshot agentmodel.MetricsSnapshot) error {
	for name, value := range snapshot.Gauges {
		rawValue := strconv.FormatFloat(value, 'f', -1, 64)
		if err := s.sendMetric(ctx, agentmodel.GaugeMetricType, name, rawValue); err != nil {
			return err
		}
	}

	for name, value := range snapshot.Counters {
		rawValue := strconv.FormatInt(value, 10)
		if err := s.sendMetric(ctx, agentmodel.CounterMetricType, name, rawValue); err != nil {
			return err
		}
	}

	return nil
}

func (s *HTTPSender) sendMetric(ctx context.Context, metricType, name, value string) error {
	metricURL := *s.baseURL
	metricURL.Path = path.Join(metricURL.Path, "update", metricType, name, value)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metricURL.String(), nil)
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
