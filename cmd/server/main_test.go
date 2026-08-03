package main

import (
	"context"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/repository"
	"j30att/observer/internal/server/storage"
)

func TestServeUntilShutdown(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	handlerFinished := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseRequest
		w.WriteHeader(http.StatusNoContent)
		close(handlerFinished)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serveUntilShutdown(ctx, server, func() error {
			return server.Serve(listener)
		}, time.Second)
	}()

	response := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			_ = resp.Body.Close()
		}
		response <- err
	}()

	<-requestStarted
	cancel()

	var shutdownErr error
	stoppedEarly := false
	select {
	case err := <-done:
		shutdownErr = err
		stoppedEarly = true
		t.Errorf("serveUntilShutdown returned before the active request completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	close(releaseRequest)
	if !stoppedEarly {
		shutdownErr = <-done
	}
	require.NoError(t, shutdownErr)
	require.NoError(t, <-response)
	select {
	case <-handlerFinished:
	default:
		require.Fail(t, "active handler did not finish")
	}
}

func TestStopPeriodicSaverPersistsFinalSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repo := repository.NewMetricsRepository()
	path := filepath.Join(t.TempDir(), "metrics.json")
	fileStorage := storage.NewFileStorage(path, zerolog.Nop())
	saver := storage.NewPeriodicSaver(time.Hour, repo, fileStorage, zerolog.Nop())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		saver.Run(ctx)
	}()

	require.NoError(t, repo.SaveGauge(context.Background(), "UnsavedGauge", 42))
	require.NoError(t, stopPeriodicSaver(cancel, &wg, saver))

	_, metrics, err := storage.NewRestoredFileStorage(path, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, metrics, 1)
	assert.Equal(t, "UnsavedGauge", metrics[0].ID)
	require.NotNil(t, metrics[0].Value)
	assert.Equal(t, float64(42), *metrics[0].Value)
}

func TestServeUntilShutdownTimesOut(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(requestStarted)
		<-releaseRequest
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &http.Server{Handler: handler, ReadHeaderTimeout: time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serveUntilShutdown(ctx, server, func() error {
			return server.Serve(listener)
		}, 20*time.Millisecond)
	}()

	response := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			_ = resp.Body.Close()
		}
		response <- err
	}()

	<-requestStarted
	cancel()
	shutdownErr := <-done
	close(releaseRequest)

	require.Error(t, shutdownErr)
	assert.ErrorIs(t, shutdownErr, context.DeadlineExceeded)
	<-response
}

func TestRunServerReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, listener.Close())
	}()

	err = runServer([]string{"-a", listener.Addr().String()}, zerolog.Nop())

	require.Error(t, err)
	assert.ErrorContains(t, err, "address already in use")
}
