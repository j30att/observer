package senders

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	agentmodel "j30att/observer/internal/agent/model"
)

func TestHTTPSenderSendsGaugeAndCounterMetrics(t *testing.T) {
	var requests []agentmodel.Metrics

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/update", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		require.Equal(t, "gzip", r.Header.Get("Accept-Encoding"))

		body := readGzipBody(t, r.Body)

		var metric agentmodel.Metrics
		require.NoError(t, json.Unmarshal(body, &metric))
		requests = append(requests, metric)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHTTPSender(server.URL)
	snapshot := agentmodel.NewMetricsSnapshot()
	snapshot.Gauges["Alloc"] = 12.5
	snapshot.Counters["PollCount"] = 7

	err := sender.Send(context.Background(), snapshot)
	require.NoError(t, err)
	require.Len(t, requests, 2)

	require.ElementsMatch(t, []string{"Alloc", "PollCount"}, []string{requests[0].ID, requests[1].ID})
	for _, metric := range requests {
		switch metric.ID {
		case "Alloc":
			require.Equal(t, agentmodel.GaugeMetricType, metric.MType)
			require.NotNil(t, metric.Value)
			require.Equal(t, 12.5, *metric.Value)
		case "PollCount":
			require.Equal(t, agentmodel.CounterMetricType, metric.MType)
			require.NotNil(t, metric.Delta)
			require.EqualValues(t, 7, *metric.Delta)
		default:
			t.Fatalf("unexpected metric id: %s", metric.ID)
		}
	}
}

func TestHTTPSenderReturnsErrorWhenServerRespondsWithUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	sender := NewHTTPSender(server.URL)
	snapshot := agentmodel.NewMetricsSnapshot()
	snapshot.Gauges["Alloc"] = 12.5

	err := sender.Send(context.Background(), snapshot)
	require.Error(t, err)
}

func TestHTTPSenderReturnsErrorWhenServerRespondsWithUnexpectedContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHTTPSender(server.URL)
	snapshot := agentmodel.NewMetricsSnapshot()
	snapshot.Gauges["Alloc"] = 12.5

	err := sender.Send(context.Background(), snapshot)
	require.Error(t, err)
}

func TestHTTPSenderAddsHTTPSchemeWhenAddressDoesNotContainIt(t *testing.T) {
	var requestedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	address := strings.TrimPrefix(server.URL, "http://")
	sender := NewHTTPSender(address)
	snapshot := agentmodel.NewMetricsSnapshot()
	snapshot.Counters["PollCount"] = 1

	err := sender.Send(context.Background(), snapshot)
	require.NoError(t, err)
	require.Equal(t, "/update", requestedPath)
}

func TestParseBaseURLTreatsLocalhostAddressAsHost(t *testing.T) {
	baseURL := parseBaseURL("localhost:8080")

	require.Equal(t, "http", baseURL.Scheme)
	require.Equal(t, "localhost:8080", baseURL.Host)
	require.Empty(t, baseURL.Path)
}

func readGzipBody(t *testing.T, body io.ReadCloser) []byte {
	t.Helper()
	defer func() {
		_ = body.Close()
	}()

	reader, err := gzip.NewReader(body)
	require.NoError(t, err)
	defer func() {
		_ = reader.Close()
	}()

	rawBody, err := io.ReadAll(reader)
	require.NoError(t, err)

	return rawBody
}
