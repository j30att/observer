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
		})

		t.Run("Должен переопределить values из flags", func(t *testing.T) {
			cfg, err := config.ParseServerConfig([]string{
				"-a=127.0.0.1:9000",
				"-i=15",
				"-f=/tmp/custom-metrics.json",
				"-r=false",
				"-d=postgres://user:password@example.com:5432/observer?sslmode=require",
			})

			require.NoError(t, err)
			assert.Equal(t, "127.0.0.1:9000", cfg.Address)
			assert.Equal(t, 15*time.Second, cfg.StoreInterval)
			assert.Equal(t, "/tmp/custom-metrics.json", cfg.FileStoragePath)
			assert.False(t, cfg.Restore)
			assert.Equal(t, "postgres://user:password@example.com:5432/observer?sslmode=require", cfg.DatabaseDSN)
		})

		t.Run("Должен переопределить values из environment", func(t *testing.T) {
			t.Setenv("ADDRESS", "127.0.0.1:9100")
			t.Setenv("STORE_INTERVAL", "20")
			t.Setenv("FILE_STORAGE_PATH", "/tmp/env-metrics.json")
			t.Setenv("RESTORE", "false")
			t.Setenv("DATABASE_DSN", "postgres://env-dsn")

			cfg, err := config.ParseServerConfig(nil)

			require.NoError(t, err)
			assert.Equal(t, "127.0.0.1:9100", cfg.Address)
			assert.Equal(t, 20*time.Second, cfg.StoreInterval)
			assert.Equal(t, "/tmp/env-metrics.json", cfg.FileStoragePath)
			assert.False(t, cfg.Restore)
			assert.Equal(t, "postgres://env-dsn", cfg.DatabaseDSN)
		})

		t.Run("Должен отдать приоритет environment над flags", func(t *testing.T) {
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
			assert.Equal(t, "127.0.0.1:9100", cfg.Address)
			assert.Equal(t, 20*time.Second, cfg.StoreInterval)
			assert.Equal(t, "/tmp/env-metrics.json", cfg.FileStoragePath)
			assert.False(t, cfg.Restore)
			assert.Equal(t, "postgres://env-dsn", cfg.DatabaseDSN)
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
	})
}
