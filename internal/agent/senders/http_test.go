package senders

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentmodel "j30att/observer/internal/agent/model"
	"j30att/observer/internal/encryption"
	"j30att/observer/internal/signature"
)

func TestHTTPSender(t *testing.T) {
	t.Run("Тест метода Send", func(t *testing.T) {
		t.Run("Должен отправить gauge и counter метрики", func(t *testing.T) {
			var request []agentmodel.Metrics
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/updates", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
				assert.Equal(t, "gzip", r.Header.Get("Accept-Encoding"))

				rawBody, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, signature.Sign(rawBody, "secret-key"), r.Header.Get(signature.Header))

				body := readGzipBody(t, io.NopCloser(bytes.NewReader(rawBody)))
				require.NoError(t, json.Unmarshal(body, &request))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			sender := NewHTTPSender(server.URL, "secret-key")
			snapshot := agentmodel.NewMetricsSnapshot()
			snapshot.Gauges["Alloc"] = 12.5
			snapshot.Counters["PollCount"] = 7

			err := sender.Send(context.Background(), snapshot)

			require.NoError(t, err)
			require.Len(t, request, 2)
			assert.ElementsMatch(t, []string{"Alloc", "PollCount"}, []string{request[0].ID, request[1].ID})
			for _, metric := range request {
				switch metric.ID {
				case "Alloc":
					assert.Equal(t, agentmodel.GaugeMetricType, metric.MType)
					require.NotNil(t, metric.Value)
					assert.Equal(t, 12.5, *metric.Value)
				case "PollCount":
					assert.Equal(t, agentmodel.CounterMetricType, metric.MType)
					require.NotNil(t, metric.Delta)
					assert.EqualValues(t, 7, *metric.Delta)
				default:
					t.Fatalf("unexpected metric id: %s", metric.ID)
				}
			}
		})

		t.Run("Должен зашифровать сжатое сообщение и подписать ciphertext", func(t *testing.T) {
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			require.NoError(t, err)

			var request []agentmodel.Metrics
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "gzip, rsa", r.Header.Get("Content-Encoding"))

				encryptedBody, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, signature.Sign(encryptedBody, "secret-key"), r.Header.Get(signature.Header))

				compressedBody, err := encryption.Decrypt(encryptedBody, privateKey)
				require.NoError(t, err)
				body := readGzipBody(t, io.NopCloser(bytes.NewReader(compressedBody)))
				require.NoError(t, json.Unmarshal(body, &request))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			sender := NewHTTPSenderWithOptions(server.URL, HTTPSenderOptions{
				SignatureKey: "secret-key",
				PublicKey:    &privateKey.PublicKey,
			})
			snapshot := agentmodel.NewMetricsSnapshot()
			snapshot.Gauges["Alloc"] = 12.5

			err = sender.Send(context.Background(), snapshot)

			require.NoError(t, err)
			require.Len(t, request, 1)
			assert.Equal(t, "Alloc", request[0].ID)
		})

		t.Run("Должен вернуть ошибку если server вернул неожиданный status", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
			}))
			defer server.Close()

			sender := NewHTTPSender(server.URL)
			snapshot := agentmodel.NewMetricsSnapshot()
			snapshot.Gauges["Alloc"] = 12.5

			err := sender.Send(context.Background(), snapshot)

			require.Error(t, err)
		})

		t.Run("Должен вернуть ошибку если server вернул неожиданный content type", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			sender := NewHTTPSender(server.URL)
			snapshot := agentmodel.NewMetricsSnapshot()
			snapshot.Gauges["Alloc"] = 12.5

			err := sender.Send(context.Background(), snapshot)

			require.Error(t, err)
		})

		t.Run("Должен добавить http scheme если address без scheme", func(t *testing.T) {
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
			assert.Equal(t, "/updates", requestedPath)
		})

		t.Run("Должен повторить отправку если соединение временно недоступно", func(t *testing.T) {
			requestsCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requestsCount++
				if requestsCount == 1 {
					conn, _, err := w.(http.Hijacker).Hijack()
					require.NoError(t, err)
					_ = conn.Close()
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			sender := NewHTTPSender(server.URL)
			sender.retryDelays = []time.Duration{0, 0, 0}
			snapshot := agentmodel.NewMetricsSnapshot()
			snapshot.Gauges["Alloc"] = 12.5

			err := sender.Send(context.Background(), snapshot)

			require.NoError(t, err)
			assert.Equal(t, 2, requestsCount)
		})

		t.Run("Не должен отправлять пустой batch", func(t *testing.T) {
			requestsCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requestsCount++
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			sender := NewHTTPSender(server.URL)

			err := sender.Send(context.Background(), agentmodel.NewMetricsSnapshot())

			require.NoError(t, err)
			assert.Zero(t, requestsCount)
		})
	})

	t.Run("Тест parseBaseURL", func(t *testing.T) {
		t.Run("Должен считать localhost address как host", func(t *testing.T) {
			baseURL := parseBaseURL("localhost:8080")

			assert.Equal(t, "http", baseURL.Scheme)
			assert.Equal(t, "localhost:8080", baseURL.Host)
			assert.Empty(t, baseURL.Path)
		})
	})
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
