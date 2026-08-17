package senders

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	agentmodel "j30att/observer/internal/agent/model"
	metricspb "j30att/observer/internal/proto"
	"j30att/observer/internal/server/grpcserver"
	"j30att/observer/internal/server/handlers/update"
	servermodel "j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

func TestGRPCSenderSend(t *testing.T) {
	repo := repository.NewMetricsRepository()
	server := grpc.NewServer(grpc.UnaryInterceptor(grpcserver.TrustedSubnetUnaryInterceptor("127.0.0.0/8")))
	metricspb.RegisterMetricsServer(server, grpcserver.NewMetricsServer(update.New(repo)))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer server.GracefulStop()
	go func() {
		_ = server.Serve(listener)
	}()

	sender, err := NewGRPCSender(listener.Addr().String())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, sender.Close())
	}()

	snapshot := agentmodel.NewMetricsSnapshot()
	snapshot.Gauges["Alloc"] = 12.5
	snapshot.Counters["PollCount"] = 7

	err = sender.Send(context.Background(), snapshot)

	require.NoError(t, err)
	metrics := repo.List(context.Background())
	require.Len(t, metrics, 2)
	for _, metric := range metrics {
		switch metric.ID {
		case "Alloc":
			assert.Equal(t, servermodel.Gauge, metric.MType)
			require.NotNil(t, metric.Value)
			assert.Equal(t, 12.5, *metric.Value)
		case "PollCount":
			assert.Equal(t, servermodel.Counter, metric.MType)
			require.NotNil(t, metric.Delta)
			assert.EqualValues(t, 7, *metric.Delta)
		default:
			t.Fatalf("unexpected metric id: %s", metric.ID)
		}
	}
}
