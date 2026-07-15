package audit_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/audit"
)

func TestFileObserver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	observer := audit.NewFileObserver(path)
	t.Cleanup(func() {
		require.NoError(t, observer.Close())
	})
	event := audit.Event{Timestamp: 12345678, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.168.0.42"}

	require.NoError(t, observer.Notify(context.Background(), event))
	require.NoError(t, observer.Notify(context.Background(), event))

	file := openFile(t, path)
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	var events []audit.Event
	for scanner.Scan() {
		var actual audit.Event
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &actual))
		events = append(events, actual)
	}
	require.NoError(t, scanner.Err())

	require.Len(t, events, 2)
	assert.Equal(t, event, events[0])
	assert.Equal(t, event, events[1])
}

func TestURLObserver(t *testing.T) {
	event := audit.Event{Timestamp: 12345678, Metrics: []string{"Alloc"}, IPAddress: "192.168.0.42"}
	received := make(chan audit.Event, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var actual audit.Event
		require.NoError(t, json.NewDecoder(r.Body).Decode(&actual))
		received <- actual

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	observer := audit.NewURLObserver(server.URL)

	require.NoError(t, observer.Notify(context.Background(), event))

	select {
	case actual := <-received:
		assert.Equal(t, event, actual)
	case <-time.After(time.Second):
		t.Fatal("audit event was not received")
	}
}

func TestURLObserverReturnsErrorOnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	observer := audit.NewURLObserver(server.URL)

	err := observer.Notify(context.Background(), audit.Event{})

	require.EqualError(t, err, "unexpected audit response status: 502 Bad Gateway")
}

func TestSubjectNotifyDoesNotBlockObservers(t *testing.T) {
	event := audit.Event{Timestamp: 12345678, Metrics: []string{"Alloc"}, IPAddress: "192.168.0.42"}
	slowStarted := make(chan struct{})
	releaseSlow := make(chan struct{})
	fastReceived := make(chan audit.Event, 1)

	slowObserver := observerFunc(func(_ context.Context, _ audit.Event) error {
		close(slowStarted)
		<-releaseSlow
		return nil
	})
	fastObserver := observerFunc(func(_ context.Context, event audit.Event) error {
		fastReceived <- event
		return nil
	})
	subject := audit.NewSubject(zerolog.Nop(), slowObserver, fastObserver)
	defer func() {
		close(releaseSlow)
		require.NoError(t, subject.Close())
	}()

	subject.Notify(context.Background(), event)

	select {
	case <-slowStarted:
	case <-time.After(time.Second):
		t.Fatal("slow observer was not called")
	}

	select {
	case actual := <-fastReceived:
		assert.Equal(t, event, actual)
	case <-time.After(time.Second):
		t.Fatal("fast observer was blocked by slow observer")
	}
}

type observerFunc func(context.Context, audit.Event) error

func (f observerFunc) Notify(ctx context.Context, event audit.Event) error {
	return f(ctx, event)
}

func openFile(t *testing.T, path string) *os.File {
	t.Helper()

	file, err := os.Open(path)
	require.NoError(t, err)

	return file
}
