package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

// AgentConfig contains runtime settings for the metrics collection agent.
type AgentConfig struct {
	ServerAddress  string
	PollInterval   time.Duration
	ReportInterval time.Duration
	Key            string
	RateLimit      int
}

// NewAgentConfig returns the default agent configuration.
func NewAgentConfig() AgentConfig {
	return AgentConfig{
		ServerAddress:  "localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		RateLimit:      1,
	}
}

// ParseAgentConfig reads agent flags and environment variables into a config.
// Environment variables override flag values.
func ParseAgentConfig(args []string) (AgentConfig, error) {
	cfg := NewAgentConfig()
	var reportSeconds int
	var pollSeconds int

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "HTTP server endpoint address")
	fs.IntVar(&reportSeconds, "r", int(cfg.ReportInterval/time.Second), "report interval in seconds")
	fs.IntVar(&pollSeconds, "p", int(cfg.PollInterval/time.Second), "poll interval in seconds")
	fs.StringVar(&cfg.Key, "k", cfg.Key, "hash signature key")
	fs.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "maximum number of concurrent outgoing requests")

	if err := fs.Parse(args); err != nil {
		return AgentConfig{}, err
	}

	if value, ok, err := lookupEnvInt("REPORT_INTERVAL"); err != nil {
		return AgentConfig{}, fmt.Errorf("invalid REPORT_INTERVAL value: %w", err)
	} else if ok {
		reportSeconds = value
	}

	if value, ok, err := lookupEnvInt("POLL_INTERVAL"); err != nil {
		return AgentConfig{}, fmt.Errorf("invalid POLL_INTERVAL value: %w", err)
	} else if ok {
		pollSeconds = value
	}

	if value, ok, err := lookupEnvInt("RATE_LIMIT"); err != nil {
		return AgentConfig{}, fmt.Errorf("invalid RATE_LIMIT value: %w", err)
	} else if ok {
		cfg.RateLimit = value
	}

	if value, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ServerAddress = value
	}

	if value, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = value
	}

	if reportSeconds < 0 {
		return AgentConfig{}, fmt.Errorf("invalid report interval value %d: interval must be non-negative seconds", reportSeconds)
	}

	if pollSeconds < 0 {
		return AgentConfig{}, fmt.Errorf("invalid poll interval value %d: interval must be non-negative seconds", pollSeconds)
	}

	if cfg.RateLimit <= 0 {
		return AgentConfig{}, fmt.Errorf("invalid rate limit value %d: value must be positive", cfg.RateLimit)
	}

	cfg.ReportInterval = time.Duration(reportSeconds) * time.Second
	cfg.PollInterval = time.Duration(pollSeconds) * time.Second

	return cfg, nil
}

func lookupEnvInt(key string) (int, bool, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return 0, false, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, true, fmt.Errorf("%q is not a valid integer", value)
	}

	return parsed, true, nil
}
