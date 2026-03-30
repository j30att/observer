package config

import "time"

type AgentConfig struct {
	ServerAddress  string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func NewAgentConfig() AgentConfig {
	return AgentConfig{
		ServerAddress:  "http://localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
	}
}
