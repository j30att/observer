package config

import (
	"flag"
	"fmt"
	"io"
	"time"
)

type AgentConfig struct {
	ServerAddress  string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func NewAgentConfig() AgentConfig {
	return AgentConfig{
		ServerAddress:  "localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
	}
}

func ParseAgentConfig(args []string) (AgentConfig, error) {
	cfg := NewAgentConfig()
	var reportSeconds int
	var pollSeconds int

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "HTTP server endpoint address")
	fs.IntVar(&reportSeconds, "r", int(cfg.ReportInterval/time.Second), "report interval in seconds")
	fs.IntVar(&pollSeconds, "p", int(cfg.PollInterval/time.Second), "poll interval in seconds")

	if err := fs.Parse(args); err != nil {
		return AgentConfig{}, err
	}

	if reportSeconds < 0 {
		return AgentConfig{}, fmt.Errorf("invalid -r value %d: interval must be non-negative seconds", reportSeconds)
	}

	if pollSeconds < 0 {
		return AgentConfig{}, fmt.Errorf("invalid -p value %d: interval must be non-negative seconds", pollSeconds)
	}

	cfg.ReportInterval = time.Duration(reportSeconds) * time.Second
	cfg.PollInterval = time.Duration(pollSeconds) * time.Second

	return cfg, nil
}
