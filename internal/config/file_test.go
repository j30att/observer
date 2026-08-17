package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/config"
)

func TestAgentFileConfig(t *testing.T) {
	t.Run("loads every option from short config flag", func(t *testing.T) {
		path := writeConfigFile(t, `{
			"address": "agent-file:8080",
			"grpc_address": "agent-file:3200",
			"report_interval": "1.5s",
			"poll_interval": "750ms",
			"key": "file-key",
			"crypto_key": "/tmp/file-public.pem",
			"rate_limit": 7
		}`)

		cfg, err := config.ParseAgentConfig([]string{"-c", path})

		require.NoError(t, err)
		assert.Equal(t, "agent-file:8080", cfg.ServerAddress)
		assert.Equal(t, "agent-file:3200", cfg.GRPCAddress)
		assert.Equal(t, 1500*time.Millisecond, cfg.ReportInterval)
		assert.Equal(t, 750*time.Millisecond, cfg.PollInterval)
		assert.Equal(t, "file-key", cfg.Key)
		assert.Equal(t, "/tmp/file-public.pem", cfg.CryptoKey)
		assert.Equal(t, 7, cfg.RateLimit)
	})

	t.Run("loads config from environment", func(t *testing.T) {
		path := writeConfigFile(t, `{"address":"agent-env-file:8080"}`)
		t.Setenv("CONFIG", path)

		cfg, err := config.ParseAgentConfig(nil)

		require.NoError(t, err)
		assert.Equal(t, "agent-env-file:8080", cfg.ServerAddress)
	})

	t.Run("flags and environment override file", func(t *testing.T) {
		path := writeConfigFile(t, `{
			"address": "from-file:8080",
			"grpc_address": "from-file:3200",
			"report_interval": "30s",
			"poll_interval": "20s",
			"crypto_key": "/tmp/file.pem"
		}`)
		t.Setenv("ADDRESS", "from-env:8080")
		t.Setenv("GRPC_ADDRESS", "from-env:3300")
		t.Setenv("POLL_INTERVAL", "3")

		cfg, err := config.ParseAgentConfig([]string{
			"-config=" + path,
			"-a=from-flag:8080",
			"-grpc-address=from-flag:3200",
			"-r=5",
			"-p=4",
			"-crypto-key=/tmp/flag.pem",
		})

		require.NoError(t, err)
		assert.Equal(t, "from-env:8080", cfg.ServerAddress)
		assert.Equal(t, "from-env:3300", cfg.GRPCAddress)
		assert.Equal(t, 5*time.Second, cfg.ReportInterval)
		assert.Equal(t, 3*time.Second, cfg.PollInterval)
		assert.Equal(t, "/tmp/flag.pem", cfg.CryptoKey)
	})
}

func TestServerFileConfig(t *testing.T) {
	t.Run("loads every option from long config flag", func(t *testing.T) {
		path := writeConfigFile(t, `{
			"address": "server-file:8080",
			"grpc_address": "server-file:3200",
			"restore": false,
			"store_interval": "1.5s",
			"store_file": "/tmp/file.db",
			"database_dsn": "postgres://file-dsn",
			"key": "file-key",
			"crypto_key": "/tmp/file-private.pem",
			"audit_file": "/tmp/file-audit.log",
			"audit_url": "https://audit.example.com/file",
			"trusted_subnet": "192.168.1.0/24"
		}`)

		cfg, err := config.ParseServerConfig([]string{"-config", path})

		require.NoError(t, err)
		assert.Equal(t, "server-file:8080", cfg.Address)
		assert.Equal(t, "server-file:3200", cfg.GRPCAddress)
		assert.False(t, cfg.Restore)
		assert.Equal(t, 1500*time.Millisecond, cfg.StoreInterval)
		assert.Equal(t, "/tmp/file.db", cfg.FileStoragePath)
		assert.Equal(t, "postgres://file-dsn", cfg.DatabaseDSN)
		assert.Equal(t, "file-key", cfg.Key)
		assert.Equal(t, "/tmp/file-private.pem", cfg.CryptoKey)
		assert.Equal(t, "/tmp/file-audit.log", cfg.AuditFile)
		assert.Equal(t, "https://audit.example.com/file", cfg.AuditURL)
		assert.Equal(t, "192.168.1.0/24", cfg.TrustedSubnet)
	})

	t.Run("CONFIG overrides config flag", func(t *testing.T) {
		flagPath := writeConfigFile(t, `{"address":"flag-file:8080"}`)
		envPath := writeConfigFile(t, `{"address":"env-file:8080"}`)
		t.Setenv("CONFIG", envPath)

		cfg, err := config.ParseServerConfig([]string{"-c", flagPath})

		require.NoError(t, err)
		assert.Equal(t, "env-file:8080", cfg.Address)
	})

	t.Run("flags and environment override file", func(t *testing.T) {
		path := writeConfigFile(t, `{
			"address": "from-file:8080",
			"grpc_address": "from-file:3200",
			"restore": true,
			"store_interval": "30s",
			"store_file": "/tmp/file.db",
			"database_dsn": "postgres://file-dsn",
			"trusted_subnet": "192.168.1.0/24"
		}`)
		t.Setenv("ADDRESS", "from-env:8080")
		t.Setenv("GRPC_ADDRESS", "from-env:3300")
		t.Setenv("STORE_INTERVAL", "3")
		t.Setenv("STORE_FILE", "/tmp/env.db")
		t.Setenv("TRUSTED_SUBNET", "10.0.0.0/8")

		cfg, err := config.ParseServerConfig([]string{
			"-c=" + path,
			"-a=from-flag:8080",
			"-grpc-address=from-flag:3200",
			"-i=5",
			"-f=/tmp/flag.db",
			"-r=false",
			"-d=postgres://flag-dsn",
			"-t=172.16.0.0/12",
		})

		require.NoError(t, err)
		assert.Equal(t, "from-env:8080", cfg.Address)
		assert.Equal(t, "from-env:3300", cfg.GRPCAddress)
		assert.Equal(t, 3*time.Second, cfg.StoreInterval)
		assert.Equal(t, "/tmp/env.db", cfg.FileStoragePath)
		assert.False(t, cfg.Restore)
		assert.Equal(t, "postgres://flag-dsn", cfg.DatabaseDSN)
		assert.Equal(t, "10.0.0.0/8", cfg.TrustedSubnet)
	})
}

func TestFileConfigErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := config.ParseAgentConfig([]string{"-c", filepath.Join(t.TempDir(), "missing.json")})

		require.Error(t, err)
		assert.ErrorContains(t, err, "read agent config file")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := writeConfigFile(t, `{`)

		_, err := config.ParseServerConfig([]string{"-c", path})

		require.Error(t, err)
		assert.ErrorContains(t, err, "parse server config file")
	})

	t.Run("invalid duration", func(t *testing.T) {
		path := writeConfigFile(t, `{"poll_interval":"soon"}`)

		_, err := config.ParseAgentConfig([]string{"-c", path})

		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid poll_interval")
	})

	t.Run("negative duration", func(t *testing.T) {
		path := writeConfigFile(t, `{"store_interval":"-1s"}`)

		_, err := config.ParseServerConfig([]string{"-c", path})

		require.EqualError(t, err, "invalid store interval value -1: interval must be non-negative seconds")
	})
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	return path
}
