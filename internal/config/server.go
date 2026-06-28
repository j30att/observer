package config

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"time"
)

// ServerConfig contains runtime settings for the metrics HTTP server.
type ServerConfig struct {
	Address         string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	Key             string
	AuditFile       string
	AuditURL        string
}

// NewServerConfig returns the default server configuration.
func NewServerConfig() ServerConfig {
	return ServerConfig{
		Address:         "localhost:8080",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "",
		Restore:         true,
	}
}

// ParseServerConfig reads server flags and environment variables into a config.
// Environment variables override flag values.
func ParseServerConfig(args []string) (ServerConfig, error) {
	cfg := NewServerConfig()
	var storeIntervalSeconds int

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Address, "a", cfg.Address, "HTTP server endpoint address")
	fs.IntVar(&storeIntervalSeconds, "i", int(cfg.StoreInterval/time.Second), "store interval in seconds")
	fs.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file storage path")
	fs.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore metrics from file storage on startup")
	fs.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database connection DSN")
	fs.StringVar(&cfg.Key, "k", cfg.Key, "hash signature key")
	fs.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "audit log file path")
	fs.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "audit log receiver URL")

	if err := fs.Parse(args); err != nil {
		return ServerConfig{}, err
	}

	if value, ok, err := lookupEnvInt("STORE_INTERVAL"); err != nil {
		return ServerConfig{}, fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
	} else if ok {
		storeIntervalSeconds = value
	}

	if value, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = value
	}

	if value, ok, err := lookupEnvBool("RESTORE"); err != nil {
		return ServerConfig{}, fmt.Errorf("invalid RESTORE value: %w", err)
	} else if ok {
		cfg.Restore = value
	}

	if value, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Address = value
	}

	if value, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = value
	}

	if value, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = value
	}

	if value, ok := os.LookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = value
	}

	if value, ok := os.LookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = value
	}

	if storeIntervalSeconds < 0 {
		return ServerConfig{}, fmt.Errorf("invalid store interval value %d: interval must be non-negative seconds", storeIntervalSeconds)
	}

	if cfg.AuditURL != "" {
		parsedURL, err := url.ParseRequestURI(cfg.AuditURL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return ServerConfig{}, fmt.Errorf("invalid audit URL %q: full URL with scheme and host is required", cfg.AuditURL)
		}
	}

	cfg.StoreInterval = time.Duration(storeIntervalSeconds) * time.Second

	return cfg, nil
}

func lookupEnvBool(key string) (bool, bool, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return false, false, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, true, fmt.Errorf("%q is not a valid boolean", value)
	}

	return parsed, true, nil
}
