package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/config"
)

func TestNewAgentConfigReturnsDefaultValues(t *testing.T) {
	cfg := config.NewAgentConfig()

	require.Equal(t, "http://localhost:8080", cfg.ServerAddress)
	require.Equal(t, 2*time.Second, cfg.PollInterval)
	require.Equal(t, 10*time.Second, cfg.ReportInterval)
}
