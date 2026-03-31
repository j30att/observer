package senders

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	agentmodel "j30att/observer/internal/agent/model"
)

func TestHTTPSenderSendsGaugeAndCounterMetrics(t *testing.T) {
	var requests []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "text/plain", r.Header.Get("Content-Type"))
		requests = append(requests, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHTTPSender(server.URL)
	snapshot := agentmodel.NewMetricsSnapshot()
	snapshot.Gauges["Alloc"] = 12.5
	snapshot.Counters["PollCount"] = 7

	err := sender.Send(context.Background(), snapshot)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"/update/gauge/Alloc/12.5",
		"/update/counter/PollCount/7",
	}, requests)
}

func TestHTTPSenderReturnsErrorWhenServerRespondsWithUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
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
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	address := strings.TrimPrefix(server.URL, "http://")
	sender := NewHTTPSender(address)
	snapshot := agentmodel.NewMetricsSnapshot()
	snapshot.Counters["PollCount"] = 1

	err := sender.Send(context.Background(), snapshot)
	require.NoError(t, err)
	require.Equal(t, "/update/counter/PollCount/1", requestedPath)
}
