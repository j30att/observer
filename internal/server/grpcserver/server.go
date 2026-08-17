package grpcserver

import (
	"context"
	"errors"

	metricspb "j30att/observer/internal/proto"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/model"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type metricsUpdater interface {
	ExecuteBatch(ctx context.Context, metrics []model.Metrics) error
}

// MetricsServer implements the protobuf Metrics service.
type MetricsServer struct {
	metricspb.UnimplementedMetricsServer
	updater metricsUpdater
}

// NewMetricsServer creates a gRPC Metrics service backed by the update use case.
func NewMetricsServer(updater metricsUpdater) *MetricsServer {
	return &MetricsServer{updater: updater}
}

// UpdateMetrics validates and stores a batch of metrics.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *metricspb.UpdateMetricsRequest) (*metricspb.UpdateMetricsResponse, error) {
	metrics := make([]model.Metrics, 0, len(req.GetMetrics()))
	for _, metric := range req.GetMetrics() {
		converted, err := convertMetric(metric)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, converted)
	}

	if err := s.updater.ExecuteBatch(ctx, metrics); err != nil {
		if errors.Is(err, update.ErrUnsupportedMetricType) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "update metrics: %v", err)
	}

	return &metricspb.UpdateMetricsResponse{}, nil
}

func convertMetric(metric *metricspb.Metric) (model.Metrics, error) {
	if metric == nil {
		return model.Metrics{}, status.Error(codes.InvalidArgument, "metric is required")
	}

	switch metric.GetType() {
	case metricspb.Metric_GAUGE:
		value := metric.GetValue()
		return model.Metrics{
			ID:    metric.GetId(),
			MType: model.Gauge,
			Value: &value,
		}, nil
	case metricspb.Metric_COUNTER:
		delta := metric.GetDelta()
		return model.Metrics{
			ID:    metric.GetId(),
			MType: model.Counter,
			Delta: &delta,
		}, nil
	default:
		return model.Metrics{}, status.Error(codes.InvalidArgument, "unsupported metric type")
	}
}
