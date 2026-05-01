package senders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	agentmodel "j30att/observer/internal/agent/model"
	"j30att/observer/internal/compression"
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
		metric := agentmodel.Metrics{
			ID:    name,
			MType: agentmodel.GaugeMetricType,
			Value: &value,
		}
		if err := s.sendMetric(ctx, metric); err != nil {
			return fmt.Errorf("send gauge metric %q: %w", name, err)
		}
	}

	for name, value := range snapshot.Counters {
		metric := agentmodel.Metrics{
			ID:    name,
			MType: agentmodel.CounterMetricType,
			Delta: &value,
		}
		if err := s.sendMetric(ctx, metric); err != nil {
			return fmt.Errorf("send counter metric %q: %w", name, err)
		}
	}

	return nil
}

func (s *HTTPSender) sendMetric(ctx context.Context, metric agentmodel.Metrics) error {
	body, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("marshal metric: %w", err)
	}

	body, err = compression.CompressGzip(body)
	if err != nil {
		return fmt.Errorf("compress metric body: %w", err)
	}

	metricURL := *s.baseURL
	metricURL.Path = strings.TrimRight(metricURL.Path, "/") + "/update"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metricURL.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create metric update request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", compression.GzipEncoding)
	req.Header.Set("Accept-Encoding", compression.GzipEncoding)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send metric update request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("unexpected content type: %s", resp.Header.Get("Content-Type"))
	}

	return nil
}
