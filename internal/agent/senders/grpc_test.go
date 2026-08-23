package senders

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	agentmodel "j30att/observer/internal/agent/model"
	metricspb "j30att/observer/internal/proto"
	"j30att/observer/internal/server/grpcserver"
	"j30att/observer/internal/server/handlers/update"
	servermodel "j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

func TestGRPCSenderSend(t *testing.T) {
	certFile, keyFile := writeSelfSignedCertificate(t)
	repo := repository.NewMetricsRepository()
	transportCredentials, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	require.NoError(t, err)
	server := grpc.NewServer(
		grpc.Creds(transportCredentials),
		grpc.UnaryInterceptor(grpcserver.TrustedSubnetUnaryInterceptor("127.0.0.0/8")),
	)
	metricspb.RegisterMetricsServer(server, grpcserver.NewMetricsServer(update.New(repo)))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer server.GracefulStop()
	go func() {
		_ = server.Serve(listener)
	}()

	sender, err := NewGRPCSender(listener.Addr().String(), GRPCSenderOptions{
		CACertFile: certFile,
		ServerName: "localhost",
	})
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

func writeSelfSignedCertificate(t *testing.T) (string, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	dir := t.TempDir()
	certFile := filepath.Join(dir, "server.crt")
	keyFile := filepath.Join(dir, "server.key")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	require.NoError(t, os.WriteFile(certFile, certPEM, 0o600))
	require.NoError(t, os.WriteFile(keyFile, keyPEM, 0o600))

	return certFile, keyFile
}
