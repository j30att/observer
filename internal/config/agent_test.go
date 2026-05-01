package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/config"
)

func TestNewAgentConfigReturnsDefaultValues(t *testing.T) {
	cfg := config.NewAgentConfig()

	require.Equal(t, "localhost:8080", cfg.ServerAddress)
	require.Equal(t, 2*time.Second, cfg.PollInterval)
	require.Equal(t, 10*time.Second, cfg.ReportInterval)
}

func TestParseAgentConfigReturnsDefaultValues(t *testing.T) {
	cfg, err := config.ParseAgentConfig(nil)

	require.NoError(t, err)
	require.Equal(t, "localhost:8080", cfg.ServerAddress)
	require.Equal(t, 2*time.Second, cfg.PollInterval)
	require.Equal(t, 10*time.Second, cfg.ReportInterval)
}

func TestParseAgentConfigOverridesFlags(t *testing.T) {
	cfg, err := config.ParseAgentConfig([]string{"-a=127.0.0.1:9000", "-r=15", "-p=5"})

	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9000", cfg.ServerAddress)
	require.Equal(t, 5*time.Second, cfg.PollInterval)
	require.Equal(t, 15*time.Second, cfg.ReportInterval)
}

func TestParseAgentConfigOverridesValuesFromEnvironment(t *testing.T) {
	t.Setenv("ADDRESS", "127.0.0.1:9100")
	t.Setenv("REPORT_INTERVAL", "20")
	t.Setenv("POLL_INTERVAL", "7")

	cfg, err := config.ParseAgentConfig(nil)

	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9100", cfg.ServerAddress)
	require.Equal(t, 7*time.Second, cfg.PollInterval)
	require.Equal(t, 20*time.Second, cfg.ReportInterval)
}

func TestParseAgentConfigEnvironmentHasPriorityOverFlags(t *testing.T) {
	t.Setenv("ADDRESS", "127.0.0.1:9100")
	t.Setenv("REPORT_INTERVAL", "20")
	t.Setenv("POLL_INTERVAL", "7")

	cfg, err := config.ParseAgentConfig([]string{"-a=127.0.0.1:9000", "-r=15", "-p=5"})

	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9100", cfg.ServerAddress)
	require.Equal(t, 7*time.Second, cfg.PollInterval)
	require.Equal(t, 20*time.Second, cfg.ReportInterval)
}

func TestParseAgentConfigReturnsErrorForUnknownFlag(t *testing.T) {
	_, err := config.ParseAgentConfig([]string{"-x=value"})

	require.EqualError(t, err, "flag provided but not defined: -x")
}

func TestParseAgentConfigReturnsErrorForNegativeReportInterval(t *testing.T) {
	_, err := config.ParseAgentConfig([]string{"-r=-1"})

	require.EqualError(t, err, "invalid report interval value -1: interval must be non-negative seconds")
}

func TestParseAgentConfigReturnsErrorForNegativePollInterval(t *testing.T) {
	_, err := config.ParseAgentConfig([]string{"-p=-1"})

	require.EqualError(t, err, "invalid poll interval value -1: interval must be non-negative seconds")
}

func TestParseAgentConfigReturnsErrorForInvalidReportIntervalEnv(t *testing.T) {
	t.Setenv("REPORT_INTERVAL", "abc")

	_, err := config.ParseAgentConfig(nil)

	require.EqualError(t, err, "invalid REPORT_INTERVAL value: \"abc\" is not a valid integer")
}

func TestParseAgentConfigReturnsErrorForNegativePollIntervalEnv(t *testing.T) {
	t.Setenv("POLL_INTERVAL", "-1")

	_, err := config.ParseAgentConfig(nil)

	require.EqualError(t, err, "invalid poll interval value -1: interval must be non-negative seconds")
}
