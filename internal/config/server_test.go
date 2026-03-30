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
