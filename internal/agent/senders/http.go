package senders

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	agentmodel "j30att/observer/internal/agent/model"
	"j30att/observer/internal/compression"
	"j30att/observer/internal/encryption"
	"j30att/observer/internal/retry"
	"j30att/observer/internal/signature"
)

// HTTPSender sends metrics to the server's HTTP batch update endpoint.
type HTTPSender struct {
	baseURL     *url.URL
	client      *http.Client
	retryDelays []time.Duration
	key         string
	publicKey   *rsa.PublicKey
}

// HTTPSenderOptions contains optional request signing and encryption settings.
type HTTPSenderOptions struct {
	SignatureKey string
	PublicKey    *rsa.PublicKey
}

const requestTimeout = 10 * time.Second

// NewHTTPSender creates an HTTP sender for the given server address.
// When a key is provided, requests are signed with HashSHA256.
func NewHTTPSender(address string, key ...string) *HTTPSender {
	var signatureKey string
	if len(key) > 0 {
		signatureKey = key[0]
	}

	return NewHTTPSenderWithOptions(address, HTTPSenderOptions{SignatureKey: signatureKey})
}

// NewHTTPSenderWithOptions creates an HTTP sender with signing and encryption settings.
func NewHTTPSenderWithOptions(address string, opts HTTPSenderOptions) *HTTPSender {
	return &HTTPSender{
		baseURL:     parseBaseURL(address),
		client:      &http.Client{Timeout: requestTimeout},
		retryDelays: retry.DefaultDelays,
		key:         opts.SignatureKey,
		publicKey:   opts.PublicKey,
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

// Send posts a metrics snapshot to the configured server.
func (s *HTTPSender) Send(ctx context.Context, snapshot agentmodel.MetricsSnapshot) error {
	metrics := make([]agentmodel.Metrics, 0, len(snapshot.Gauges)+len(snapshot.Counters))
	gaugeValues := make([]float64, len(snapshot.Gauges))
	counterValues := make([]int64, len(snapshot.Counters))

	i := 0
	for name, value := range snapshot.Gauges {
		gaugeValues[i] = value
		metrics = append(metrics, agentmodel.Metrics{
			ID:    name,
			MType: agentmodel.GaugeMetricType,
			Value: &gaugeValues[i],
		})
		i++
	}

	i = 0
	for name, value := range snapshot.Counters {
		counterValues[i] = value
		metrics = append(metrics, agentmodel.Metrics{
			ID:    name,
			MType: agentmodel.CounterMetricType,
			Delta: &counterValues[i],
		})
		i++
	}

	if len(metrics) == 0 {
		return nil
	}

	return s.sendMetrics(ctx, metrics)
}

func (s *HTTPSender) sendMetrics(ctx context.Context, metrics []agentmodel.Metrics) error {
	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}

	body, err = compression.CompressGzip(body)
	if err != nil {
		return fmt.Errorf("compress metrics body: %w", err)
	}
	if s.publicKey != nil {
		body, err = encryption.Encrypt(body, s.publicKey)
		if err != nil {
			return fmt.Errorf("encrypt metrics body: %w", err)
		}
	}

	metricURL := *s.baseURL
	metricURL.Path = strings.TrimRight(metricURL.Path, "/") + "/updates"

	return retry.Do(ctx, s.retryDelays, isRetriableTransportError, func() error {
		return s.doSendMetrics(ctx, metricURL.String(), body)
	})
}

func (s *HTTPSender) doSendMetrics(ctx context.Context, metricURL string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metricURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create metrics update request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	contentEncoding := compression.GzipEncoding
	if s.publicKey != nil {
		contentEncoding += ", " + encryption.Encoding
	}
	req.Header.Set("Content-Encoding", contentEncoding)
	req.Header.Set("Accept-Encoding", compression.GzipEncoding)
	if realIP, err := outboundIP(ctx, metricURL); err == nil {
		req.Header.Set("X-Real-IP", realIP)
	}
	if s.key != "" {
		req.Header.Set(signature.Header, signature.Sign(body, s.key))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send metrics update request: %w", err)
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

func outboundIP(ctx context.Context, rawURL string) (string, error) {
	metricURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	address := metricURL.Host
	if _, _, err := net.SplitHostPort(address); err != nil {
		address = net.JoinHostPort(address, "80")
	}

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "udp", address)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = conn.Close()
	}()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || addr.IP == nil {
		return "", fmt.Errorf("unexpected local address: %s", conn.LocalAddr())
	}

	return addr.IP.String(), nil
}

func isRetriableTransportError(err error) bool {
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
