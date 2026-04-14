package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/config"
)

func TestNewServerConfigReturnsDefaultValues(t *testing.T) {
	cfg := config.NewServerConfig()

	require.Equal(t, "localhost:8080", cfg.Address)
}

func TestParseServerConfigReturnsDefaultValues(t *testing.T) {
	cfg, err := config.ParseServerConfig(nil)

	require.NoError(t, err)
	require.Equal(t, "localhost:8080", cfg.Address)
}

func TestParseServerConfigOverridesAddress(t *testing.T) {
	cfg, err := config.ParseServerConfig([]string{"-a=127.0.0.1:9000"})

	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9000", cfg.Address)
}

func TestParseServerConfigOverridesAddressFromEnvironment(t *testing.T) {
	t.Setenv("ADDRESS", "127.0.0.1:9100")

	cfg, err := config.ParseServerConfig(nil)

	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9100", cfg.Address)
}

func TestParseServerConfigEnvironmentHasPriorityOverFlag(t *testing.T) {
	t.Setenv("ADDRESS", "127.0.0.1:9100")

	cfg, err := config.ParseServerConfig([]string{"-a=127.0.0.1:9000"})

	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9100", cfg.Address)
}

func TestParseServerConfigReturnsErrorForUnknownFlag(t *testing.T) {
	_, err := config.ParseServerConfig([]string{"-x=value"})

	require.EqualError(t, err, "flag provided but not defined: -x")
}
