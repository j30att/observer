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

func TestParseAgentConfigReturnsErrorForUnknownFlag(t *testing.T) {
	_, err := config.ParseAgentConfig([]string{"-x=value"})

	require.EqualError(t, err, "flag provided but not defined: -x")
}

func TestParseAgentConfigReturnsErrorForNegativeReportInterval(t *testing.T) {
	_, err := config.ParseAgentConfig([]string{"-r=-1"})

	require.EqualError(t, err, "invalid -r value -1: interval must be non-negative seconds")
}

func TestParseAgentConfigReturnsErrorForNegativePollInterval(t *testing.T) {
	_, err := config.ParseAgentConfig([]string{"-p=-1"})

	require.EqualError(t, err, "invalid -p value -1: interval must be non-negative seconds")
}
