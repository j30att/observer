package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/config"
)

func TestNewServerConfigReturnsDefaultValues(t *testing.T) {
	cfg := config.NewServerConfig()

	require.Equal(t, "localhost:8080", cfg.Address)
	require.Equal(t, 300*time.Second, cfg.StoreInterval)
	require.Equal(t, "/tmp/metrics-db.json", cfg.FileStoragePath)
	require.True(t, cfg.Restore)
	require.Empty(t, cfg.DatabaseDSN)
}

func TestParseServerConfigReturnsDefaultValues(t *testing.T) {
	cfg, err := config.ParseServerConfig(nil)

	require.NoError(t, err)
	require.Equal(t, "localhost:8080", cfg.Address)
	require.Equal(t, 300*time.Second, cfg.StoreInterval)
	require.Equal(t, "/tmp/metrics-db.json", cfg.FileStoragePath)
	require.True(t, cfg.Restore)
	require.Empty(t, cfg.DatabaseDSN)
}

func TestParseServerConfigOverridesFlags(t *testing.T) {
	cfg, err := config.ParseServerConfig([]string{
		"-a=127.0.0.1:9000",
		"-i=15",
		"-f=/tmp/custom-metrics.json",
		"-r=false",
		"-d=postgres://user:password@example.com:5432/observer?sslmode=require",
	})

	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9000", cfg.Address)
	require.Equal(t, 15*time.Second, cfg.StoreInterval)
	require.Equal(t, "/tmp/custom-metrics.json", cfg.FileStoragePath)
	require.False(t, cfg.Restore)
	require.Equal(t, "postgres://user:password@example.com:5432/observer?sslmode=require", cfg.DatabaseDSN)
}

func TestParseServerConfigOverridesValuesFromEnvironment(t *testing.T) {
	t.Setenv("ADDRESS", "127.0.0.1:9100")
	t.Setenv("STORE_INTERVAL", "20")
	t.Setenv("FILE_STORAGE_PATH", "/tmp/env-metrics.json")
	t.Setenv("RESTORE", "false")
	t.Setenv("DATABASE_DSN", "postgres://env-dsn")

	cfg, err := config.ParseServerConfig(nil)

	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9100", cfg.Address)
	require.Equal(t, 20*time.Second, cfg.StoreInterval)
	require.Equal(t, "/tmp/env-metrics.json", cfg.FileStoragePath)
	require.False(t, cfg.Restore)
	require.Equal(t, "postgres://env-dsn", cfg.DatabaseDSN)
}

func TestParseServerConfigEnvironmentHasPriorityOverFlags(t *testing.T) {
	t.Setenv("ADDRESS", "127.0.0.1:9100")
	t.Setenv("STORE_INTERVAL", "20")
	t.Setenv("FILE_STORAGE_PATH", "/tmp/env-metrics.json")
	t.Setenv("RESTORE", "false")
	t.Setenv("DATABASE_DSN", "postgres://env-dsn")

	cfg, err := config.ParseServerConfig([]string{
		"-a=127.0.0.1:9000",
		"-i=15",
		"-f=/tmp/custom-metrics.json",
		"-r=true",
		"-d=postgres://flag-dsn",
	})

	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9100", cfg.Address)
	require.Equal(t, 20*time.Second, cfg.StoreInterval)
	require.Equal(t, "/tmp/env-metrics.json", cfg.FileStoragePath)
	require.False(t, cfg.Restore)
	require.Equal(t, "postgres://env-dsn", cfg.DatabaseDSN)
}

func TestParseServerConfigReturnsErrorForUnknownFlag(t *testing.T) {
	_, err := config.ParseServerConfig([]string{"-x=value"})

	require.EqualError(t, err, "flag provided but not defined: -x")
}

func TestParseServerConfigReturnsErrorForNegativeStoreInterval(t *testing.T) {
	_, err := config.ParseServerConfig([]string{"-i=-1"})

	require.EqualError(t, err, "invalid store interval value -1: interval must be non-negative seconds")
}

func TestParseServerConfigReturnsErrorForInvalidStoreIntervalEnv(t *testing.T) {
	t.Setenv("STORE_INTERVAL", "abc")

	_, err := config.ParseServerConfig(nil)

	require.EqualError(t, err, "invalid STORE_INTERVAL value: \"abc\" is not a valid integer")
}

func TestParseServerConfigReturnsErrorForNegativeStoreIntervalEnv(t *testing.T) {
	t.Setenv("STORE_INTERVAL", "-1")

	_, err := config.ParseServerConfig(nil)

	require.EqualError(t, err, "invalid store interval value -1: interval must be non-negative seconds")
}

func TestParseServerConfigReturnsErrorForInvalidRestoreEnv(t *testing.T) {
	t.Setenv("RESTORE", "maybe")

	_, err := config.ParseServerConfig(nil)

	require.EqualError(t, err, "invalid RESTORE value: \"maybe\" is not a valid boolean")
}
