package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/config"
)

func TestServerConfig(t *testing.T) {
	t.Run("Тест создания config", func(t *testing.T) {
		t.Run("Должен вернуть default values", func(t *testing.T) {
			cfg := config.NewServerConfig()

			assert.Equal(t, "localhost:8080", cfg.Address)
			assert.Equal(t, 300*time.Second, cfg.StoreInterval)
			assert.Empty(t, cfg.FileStoragePath)
			assert.True(t, cfg.Restore)
			assert.Empty(t, cfg.DatabaseDSN)
			assert.Empty(t, cfg.Key)
			assert.Empty(t, cfg.AuditFile)
			assert.Empty(t, cfg.AuditURL)
		})
	})

	t.Run("Тест парсинга config", func(t *testing.T) {
		t.Run("Должен вернуть default values", func(t *testing.T) {
			cfg, err := config.ParseServerConfig(nil)

			require.NoError(t, err)
			assert.Equal(t, "localhost:8080", cfg.Address)
			assert.Equal(t, 300*time.Second, cfg.StoreInterval)
			assert.Empty(t, cfg.FileStoragePath)
			assert.True(t, cfg.Restore)
			assert.Empty(t, cfg.DatabaseDSN)
			assert.Empty(t, cfg.Key)
			assert.Empty(t, cfg.AuditFile)
			assert.Empty(t, cfg.AuditURL)
		})

		t.Run("Должен переопределить values из flags", func(t *testing.T) {
			cfg, err := config.ParseServerConfig([]string{
				"-a=127.0.0.1:9000",
				"-i=15",
				"-f=/tmp/custom-metrics.json",
				"-r=false",
				"-d=postgres://user:password@example.com:5432/observer?sslmode=require",
				"-k=flag-key",
				"--audit-file=/tmp/audit.log",
				"--audit-url=https://audit.example.com/events",
			})

			require.NoError(t, err)
			assert.Equal(t, "127.0.0.1:9000", cfg.Address)
			assert.Equal(t, 15*time.Second, cfg.StoreInterval)
			assert.Equal(t, "/tmp/custom-metrics.json", cfg.FileStoragePath)
			assert.False(t, cfg.Restore)
			assert.Equal(t, "postgres://user:password@example.com:5432/observer?sslmode=require", cfg.DatabaseDSN)
			assert.Equal(t, "flag-key", cfg.Key)
			assert.Equal(t, "/tmp/audit.log", cfg.AuditFile)
			assert.Equal(t, "https://audit.example.com/events", cfg.AuditURL)
		})

		t.Run("Должен переопределить values из environment", func(t *testing.T) {
			t.Setenv("ADDRESS", "127.0.0.1:9100")
			t.Setenv("STORE_INTERVAL", "20")
			t.Setenv("FILE_STORAGE_PATH", "/tmp/env-metrics.json")
			t.Setenv("RESTORE", "false")
			t.Setenv("DATABASE_DSN", "postgres://env-dsn")
			t.Setenv("KEY", "env-key")
			t.Setenv("AUDIT_FILE", "/tmp/env-audit.log")
			t.Setenv("AUDIT_URL", "https://audit.example.com/env")

			cfg, err := config.ParseServerConfig(nil)

			require.NoError(t, err)
			assert.Equal(t, "127.0.0.1:9100", cfg.Address)
			assert.Equal(t, 20*time.Second, cfg.StoreInterval)
			assert.Equal(t, "/tmp/env-metrics.json", cfg.FileStoragePath)
			assert.False(t, cfg.Restore)
			assert.Equal(t, "postgres://env-dsn", cfg.DatabaseDSN)
			assert.Equal(t, "env-key", cfg.Key)
			assert.Equal(t, "/tmp/env-audit.log", cfg.AuditFile)
			assert.Equal(t, "https://audit.example.com/env", cfg.AuditURL)
		})

		t.Run("Должен отдать приоритет environment над flags", func(t *testing.T) {
			t.Setenv("ADDRESS", "127.0.0.1:9100")
			t.Setenv("STORE_INTERVAL", "20")
			t.Setenv("FILE_STORAGE_PATH", "/tmp/env-metrics.json")
			t.Setenv("RESTORE", "false")
			t.Setenv("DATABASE_DSN", "postgres://env-dsn")
			t.Setenv("KEY", "env-key")
			t.Setenv("AUDIT_FILE", "/tmp/env-audit.log")
			t.Setenv("AUDIT_URL", "https://audit.example.com/env")

			cfg, err := config.ParseServerConfig([]string{
				"-a=127.0.0.1:9000",
				"-i=15",
				"-f=/tmp/custom-metrics.json",
				"-r=true",
				"-d=postgres://flag-dsn",
				"-k=flag-key",
				"--audit-file=/tmp/flag-audit.log",
				"--audit-url=https://audit.example.com/flag",
			})

			require.NoError(t, err)
			assert.Equal(t, "127.0.0.1:9100", cfg.Address)
			assert.Equal(t, 20*time.Second, cfg.StoreInterval)
			assert.Equal(t, "/tmp/env-metrics.json", cfg.FileStoragePath)
			assert.False(t, cfg.Restore)
			assert.Equal(t, "postgres://env-dsn", cfg.DatabaseDSN)
			assert.Equal(t, "env-key", cfg.Key)
			assert.Equal(t, "/tmp/env-audit.log", cfg.AuditFile)
			assert.Equal(t, "https://audit.example.com/env", cfg.AuditURL)
		})

		t.Run("Ошибка, неизвестный flag", func(t *testing.T) {
			_, err := config.ParseServerConfig([]string{"-x=value"})

			require.EqualError(t, err, "flag provided but not defined: -x")
		})

		t.Run("Ошибка, отрицательный store interval", func(t *testing.T) {
			_, err := config.ParseServerConfig([]string{"-i=-1"})

			require.EqualError(t, err, "invalid store interval value -1: interval must be non-negative seconds")
		})

		t.Run("Ошибка, невалидный STORE_INTERVAL", func(t *testing.T) {
			t.Setenv("STORE_INTERVAL", "abc")

			_, err := config.ParseServerConfig(nil)

			require.EqualError(t, err, "invalid STORE_INTERVAL value: \"abc\" is not a valid integer")
		})

		t.Run("Ошибка, отрицательный STORE_INTERVAL", func(t *testing.T) {
			t.Setenv("STORE_INTERVAL", "-1")

			_, err := config.ParseServerConfig(nil)

			require.EqualError(t, err, "invalid store interval value -1: interval must be non-negative seconds")
		})

		t.Run("Ошибка, невалидный RESTORE", func(t *testing.T) {
			t.Setenv("RESTORE", "maybe")

			_, err := config.ParseServerConfig(nil)

			require.EqualError(t, err, "invalid RESTORE value: \"maybe\" is not a valid boolean")
		})

		t.Run("Ошибка, невалидный audit URL", func(t *testing.T) {
			_, err := config.ParseServerConfig([]string{"--audit-url=localhost:9000/audit"})

			require.EqualError(t, err, `invalid audit URL "localhost:9000/audit": full URL with scheme and host is required`)
		})
	})
}
