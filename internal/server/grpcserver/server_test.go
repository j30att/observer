package grpcserver_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	metricspb "j30att/observer/internal/proto"
	"j30att/observer/internal/server/grpcserver"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

func TestMetricsServerUpdateMetrics(t *testing.T) {
	repo := repository.NewMetricsRepository()
	server := grpcserver.NewMetricsServer(update.New(repo))

	_, err := server.UpdateMetrics(context.Background(), metricspb.UpdateMetricsRequest_builder{
		Metrics: []*metricspb.Metric{
			metricspb.Metric_builder{Id: "Alloc", Type: metricspb.Metric_GAUGE, Value: 12.5}.Build(),
			metricspb.Metric_builder{Id: "PollCount", Type: metricspb.Metric_COUNTER, Delta: 3}.Build(),
		},
	}.Build())

	require.NoError(t, err)
	metrics := repo.List(context.Background())
	require.Len(t, metrics, 2)
	assert.Equal(t, model.Gauge, metrics[0].MType)
	require.NotNil(t, metrics[0].Value)
	assert.Equal(t, 12.5, *metrics[0].Value)
	assert.Equal(t, model.Counter, metrics[1].MType)
	require.NotNil(t, metrics[1].Delta)
	assert.Equal(t, int64(3), *metrics[1].Delta)
}

func TestTrustedSubnetUnaryInterceptor(t *testing.T) {
	interceptor := grpcserver.TrustedSubnetUnaryInterceptor("192.168.1.0/24")
	handler := func(context.Context, any) (any, error) {
		return "ok", nil
	}

	t.Run("allows trusted ip", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "192.168.1.42"))

		resp, err := interceptor(ctx, nil, nil, handler)

		require.NoError(t, err)
		assert.Equal(t, "ok", resp)
	})

	t.Run("denies untrusted ip", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "10.0.0.42"))

		_, err := interceptor(ctx, nil, nil, handler)

		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})
}
