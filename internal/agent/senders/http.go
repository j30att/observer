package senders

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
		metric := agentmodel.Metrics{
			ID:    name,
			MType: agentmodel.GaugeMetricType,
			Value: &value,
		}
		if err := s.sendMetric(ctx, metric); err != nil {
			return err
		}
	}

	for name, value := range snapshot.Counters {
		metric := agentmodel.Metrics{
			ID:    name,
			MType: agentmodel.CounterMetricType,
			Delta: &value,
		}
		if err := s.sendMetric(ctx, metric); err != nil {
			return err
		}
	}

	return nil
}

func (s *HTTPSender) sendMetric(ctx context.Context, metric agentmodel.Metrics) error {
	body, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	body, err = compressBody(body)
	if err != nil {
		return err
	}

	metricURL := *s.baseURL
	metricURL.Path = strings.TrimRight(metricURL.Path, "/") + "/update"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metricURL.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

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

	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("unexpected content type: %s", resp.Header.Get("Content-Type"))
	}

	return nil
}

func compressBody(body []byte) ([]byte, error) {
	var compressed bytes.Buffer

	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return compressed.Bytes(), nil
}
