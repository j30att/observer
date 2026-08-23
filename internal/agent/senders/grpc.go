package senders

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"

	agentmodel "j30att/observer/internal/agent/model"
	metricspb "j30att/observer/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const realIPMetadataKey = "x-real-ip"

// GRPCSender sends metrics to the server's gRPC batch update endpoint.
type GRPCSender struct {
	address string
	conn    *grpc.ClientConn
	client  metricspb.MetricsClient
}

// GRPCSenderOptions contains TLS settings for the gRPC sender.
type GRPCSenderOptions struct {
	CACertFile string
	ServerName string
}

// NewGRPCSender creates a gRPC sender for the given server address.
func NewGRPCSender(address string, opts GRPCSenderOptions) (*GRPCSender, error) {
	transportCredentials, err := newClientTLSCredentials(opts)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return nil, fmt.Errorf("create grpc client: %w", err)
	}

	return &GRPCSender{
		address: address,
		conn:    conn,
		client:  metricspb.NewMetricsClient(conn),
	}, nil
}

func newClientTLSCredentials(opts GRPCSenderOptions) (credentials.TransportCredentials, error) {
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("load system certificate pool: %w", err)
	}
	if roots == nil {
		roots = x509.NewCertPool()
	}

	if opts.CACertFile != "" {
		cert, err := os.ReadFile(opts.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("read grpc CA certificate: %w", err)
		}
		if ok := roots.AppendCertsFromPEM(cert); !ok {
			return nil, fmt.Errorf("parse grpc CA certificate %q: no certificates found", opts.CACertFile)
		}
	}

	return credentials.NewTLS(&tls.Config{
		RootCAs:    roots,
		ServerName: opts.ServerName,
		MinVersion: tls.VersionTLS12,
	}), nil
}

// Send posts a metrics snapshot to the configured gRPC server.
func (s *GRPCSender) Send(ctx context.Context, snapshot agentmodel.MetricsSnapshot) error {
	metrics := make([]*metricspb.Metric, 0, len(snapshot.Gauges)+len(snapshot.Counters))
	for name, value := range snapshot.Gauges {
		metrics = append(metrics, metricspb.Metric_builder{
			Id:    name,
			Type:  metricspb.Metric_GAUGE,
			Value: value,
		}.Build())
	}
	for name, delta := range snapshot.Counters {
		metrics = append(metrics, metricspb.Metric_builder{
			Id:    name,
			Type:  metricspb.Metric_COUNTER,
			Delta: delta,
		}.Build())
	}

	if len(metrics) == 0 {
		return nil
	}

	requestCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	if realIP, err := outboundIPForAddress(requestCtx, s.address); err == nil {
		requestCtx = metadata.AppendToOutgoingContext(requestCtx, realIPMetadataKey, realIP)
	}

	_, err := s.client.UpdateMetrics(requestCtx, metricspb.UpdateMetricsRequest_builder{Metrics: metrics}.Build())
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
