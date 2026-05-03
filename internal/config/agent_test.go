package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/config"
)

func TestAgentConfig(t *testing.T) {
	t.Run("Тест создания config", func(t *testing.T) {
		t.Run("Должен вернуть default values", func(t *testing.T) {
			cfg := config.NewAgentConfig()

			assert.Equal(t, "localhost:8080", cfg.ServerAddress)
			assert.Equal(t, 2*time.Second, cfg.PollInterval)
			assert.Equal(t, 10*time.Second, cfg.ReportInterval)
		})
	})

	t.Run("Тест парсинга config", func(t *testing.T) {
		t.Run("Должен вернуть default values", func(t *testing.T) {
			cfg, err := config.ParseAgentConfig(nil)

			require.NoError(t, err)
			assert.Equal(t, "localhost:8080", cfg.ServerAddress)
			assert.Equal(t, 2*time.Second, cfg.PollInterval)
			assert.Equal(t, 10*time.Second, cfg.ReportInterval)
		})

		t.Run("Должен переопределить values из flags", func(t *testing.T) {
			cfg, err := config.ParseAgentConfig([]string{"-a=127.0.0.1:9000", "-r=15", "-p=5"})

			require.NoError(t, err)
			assert.Equal(t, "127.0.0.1:9000", cfg.ServerAddress)
			assert.Equal(t, 5*time.Second, cfg.PollInterval)
			assert.Equal(t, 15*time.Second, cfg.ReportInterval)
		})

		t.Run("Должен переопределить values из environment", func(t *testing.T) {
			t.Setenv("ADDRESS", "127.0.0.1:9100")
			t.Setenv("REPORT_INTERVAL", "20")
			t.Setenv("POLL_INTERVAL", "7")

			cfg, err := config.ParseAgentConfig(nil)

			require.NoError(t, err)
			assert.Equal(t, "127.0.0.1:9100", cfg.ServerAddress)
			assert.Equal(t, 7*time.Second, cfg.PollInterval)
			assert.Equal(t, 20*time.Second, cfg.ReportInterval)
		})

		t.Run("Должен отдать приоритет environment над flags", func(t *testing.T) {
			t.Setenv("ADDRESS", "127.0.0.1:9100")
			t.Setenv("REPORT_INTERVAL", "20")
			t.Setenv("POLL_INTERVAL", "7")

			cfg, err := config.ParseAgentConfig([]string{"-a=127.0.0.1:9000", "-r=15", "-p=5"})

			require.NoError(t, err)
			assert.Equal(t, "127.0.0.1:9100", cfg.ServerAddress)
			assert.Equal(t, 7*time.Second, cfg.PollInterval)
			assert.Equal(t, 20*time.Second, cfg.ReportInterval)
		})

		t.Run("Ошибка, неизвестный flag", func(t *testing.T) {
			_, err := config.ParseAgentConfig([]string{"-x=value"})

			require.EqualError(t, err, "flag provided but not defined: -x")
		})

		t.Run("Ошибка, отрицательный report interval", func(t *testing.T) {
			_, err := config.ParseAgentConfig([]string{"-r=-1"})

			require.EqualError(t, err, "invalid report interval value -1: interval must be non-negative seconds")
		})

		t.Run("Ошибка, отрицательный poll interval", func(t *testing.T) {
			_, err := config.ParseAgentConfig([]string{"-p=-1"})

			require.EqualError(t, err, "invalid poll interval value -1: interval must be non-negative seconds")
		})

		t.Run("Ошибка, невалидный REPORT_INTERVAL", func(t *testing.T) {
			t.Setenv("REPORT_INTERVAL", "abc")

			_, err := config.ParseAgentConfig(nil)

			require.EqualError(t, err, "invalid REPORT_INTERVAL value: \"abc\" is not a valid integer")
		})

		t.Run("Ошибка, отрицательный POLL_INTERVAL", func(t *testing.T) {
			t.Setenv("POLL_INTERVAL", "-1")

			_, err := config.ParseAgentConfig(nil)

			require.EqualError(t, err, "invalid poll interval value -1: interval must be non-negative seconds")
		})
	})
}
