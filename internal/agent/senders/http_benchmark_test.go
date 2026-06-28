package senders

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"j30att/observer/internal/agent/model"
)

func BenchmarkHTTPSenderSend(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHTTPSender(server.URL)
	snapshot := model.NewMetricsSnapshot()
	for i := range 100 {
		snapshot.Gauges["gauge_"+strconv.Itoa(i)] = float64(i)
		snapshot.Counters["counter_"+strconv.Itoa(i)] = int64(i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if err := sender.Send(context.Background(), snapshot); err != nil {
			b.Fatal(err)
		}
	}
}
