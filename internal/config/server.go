package config

import (
	"flag"
	"io"
	"os"
)

type ServerConfig struct {
	Address string
}

func NewServerConfig() ServerConfig {
	return ServerConfig{
		Address: "localhost:8080",
	}
}

func ParseServerConfig(args []string) (ServerConfig, error) {
	cfg := NewServerConfig()

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Address, "a", cfg.Address, "HTTP server endpoint address")

	if err := fs.Parse(args); err != nil {
		return ServerConfig{}, err
	}

	if value, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Address = value
	}

	return cfg, nil
}
