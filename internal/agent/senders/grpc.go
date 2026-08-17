package senders

import (
	"context"
	"fmt"
	"net"

	agentmodel "j30att/observer/internal/agent/model"
	metricspb "j30att/observer/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const realIPMetadataKey = "x-real-ip"

// GRPCSender sends metrics to the server's gRPC batch update endpoint.
type GRPCSender struct {
	address string
	conn    *grpc.ClientConn
	client  metricspb.MetricsClient
}

// NewGRPCSender creates a gRPC sender for the given server address.
func NewGRPCSender(address string) (*GRPCSender, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create grpc client: %w", err)
	}

	return &GRPCSender{
		address: address,
		conn:    conn,
		client:  metricspb.NewMetricsClient(conn),
	}, nil
}

// Send posts a metrics snapshot to the configured gRPC server.
func (s *GRPCSender) Send(ctx context.Context, snapshot agentmodel.MetricsSnapshot) error {
	metrics := make([]*metricspb.Metric, 0, len(snapshot.Gauges)+len(snapshot.Counters))
	for name, value := range snapshot.Gauges {
		metrics = append(metrics, &metricspb.Metric{
			Id:    name,
			Type:  metricspb.Metric_GAUGE,
			Value: value,
		})
	}
	for name, delta := range snapshot.Counters {
		metrics = append(metrics, &metricspb.Metric{
			Id:    name,
			Type:  metricspb.Metric_COUNTER,
			Delta: delta,
		})
	}

	if len(metrics) == 0 {
		return nil
	}

	requestCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	if realIP, err := outboundIPForAddress(requestCtx, s.address); err == nil {
		requestCtx = metadata.AppendToOutgoingContext(requestCtx, realIPMetadataKey, realIP)
	}

	_, err := s.client.UpdateMetrics(requestCtx, &metricspb.UpdateMetricsRequest{Metrics: metrics})
	if err != nil {
		return fmt.Errorf("send grpc metrics update request: %w", err)
	}

	return nil
}

// Close closes the underlying gRPC client connection.
func (s *GRPCSender) Close() error {
	return s.conn.Close()
}

func outboundIPForAddress(ctx context.Context, address string) (string, error) {
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
