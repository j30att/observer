package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ServerConfig contains runtime settings for the metrics HTTP server.
type ServerConfig struct {
	Address         string
	GRPCAddress     string
	GRPCCertFile    string
	GRPCKeyFile     string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	Key             string
	CryptoKey       string
	AuditFile       string
	AuditURL        string
	TrustedSubnet   string
}

type serverFileConfig struct {
	Address       *string `json:"address"`
	GRPCAddress   *string `json:"grpc_address"`
	GRPCCertFile  *string `json:"grpc_cert_file"`
	GRPCKeyFile   *string `json:"grpc_key_file"`
	Restore       *bool   `json:"restore"`
	StoreInterval *string `json:"store_interval"`
	StoreFile     *string `json:"store_file"`
	DatabaseDSN   *string `json:"database_dsn"`
	Key           *string `json:"key"`
	CryptoKey     *string `json:"crypto_key"`
	AuditFile     *string `json:"audit_file"`
	AuditURL      *string `json:"audit_url"`
	TrustedSubnet *string `json:"trusted_subnet"`
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

// ParseServerConfig reads the server config file, flags, and environment variables.
// Environment variables override flags, and flags override values from the file.
func ParseServerConfig(args []string) (ServerConfig, error) {
	probe := NewServerConfig()
	configPath, err := parseServerFlags(&probe, args)
	if err != nil {
		return ServerConfig{}, err
	}
	if value, ok := os.LookupEnv("CONFIG"); ok {
		configPath = value
	}

	cfg := NewServerConfig()
	if configPath != "" {
		if err := loadServerFile(configPath, &cfg); err != nil {
			return ServerConfig{}, err
		}
	}

	if _, err := parseServerFlags(&cfg, args); err != nil {
		return ServerConfig{}, err
	}

	if value, ok, err := lookupEnvDurationSeconds("STORE_INTERVAL"); err != nil {
		return ServerConfig{}, fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
	} else if ok {
		cfg.StoreInterval = value
	}

	if value, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = value
	}

	if value, ok := os.LookupEnv("STORE_FILE"); ok {
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

	if value, ok := os.LookupEnv("GRPC_ADDRESS"); ok {
		cfg.GRPCAddress = value
	}

	if value, ok := os.LookupEnv("GRPC_CERT_FILE"); ok {
		cfg.GRPCCertFile = value
	}

	if value, ok := os.LookupEnv("GRPC_KEY_FILE"); ok {
		cfg.GRPCKeyFile = value
	}

	if value, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = value
	}

	if value, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = value
	}

	if value, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = value
	}

	if value, ok := os.LookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = value
	}

	if value, ok := os.LookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = value
	}

	if value, ok := os.LookupEnv("TRUSTED_SUBNET"); ok {
		cfg.TrustedSubnet = value
	}

	if cfg.StoreInterval < 0 {
		return ServerConfig{}, fmt.Errorf("invalid store interval value %d: interval must be non-negative seconds", int(cfg.StoreInterval/time.Second))
	}

	if cfg.AuditURL != "" {
		parsedURL, err := url.ParseRequestURI(cfg.AuditURL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return ServerConfig{}, fmt.Errorf("invalid audit URL %q: full URL with scheme and host is required", cfg.AuditURL)
		}
	}

	cfg.TrustedSubnet = strings.TrimSpace(cfg.TrustedSubnet)
	if cfg.TrustedSubnet != "" {
		if _, err := netip.ParsePrefix(cfg.TrustedSubnet); err != nil {
			return ServerConfig{}, fmt.Errorf("invalid trusted subnet %q: %w", cfg.TrustedSubnet, err)
		}
	}

	return cfg, nil
}

func parseServerFlags(cfg *ServerConfig, args []string) (string, error) {
	var storeIntervalSeconds int
	var configPath string

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Address, "a", cfg.Address, "HTTP server endpoint address")
	fs.StringVar(&cfg.GRPCAddress, "grpc-address", cfg.GRPCAddress, "gRPC server endpoint address")
	fs.StringVar(&cfg.GRPCCertFile, "grpc-cert", cfg.GRPCCertFile, "path to the gRPC TLS certificate file")
	fs.StringVar(&cfg.GRPCKeyFile, "grpc-key", cfg.GRPCKeyFile, "path to the gRPC TLS private key file")
	fs.IntVar(&storeIntervalSeconds, "i", int(cfg.StoreInterval/time.Second), "store interval in seconds")
	fs.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file storage path")
	fs.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore metrics from file storage on startup")
	fs.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database connection DSN")
	fs.StringVar(&cfg.Key, "k", cfg.Key, "hash signature key")
	fs.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "path to the private encryption key")
	fs.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "audit log file path")
	fs.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "audit log receiver URL")
	fs.StringVar(&cfg.TrustedSubnet, "t", cfg.TrustedSubnet, "trusted agent subnet in CIDR notation")
	fs.StringVar(&configPath, "c", configPath, "path to the JSON configuration file")
	fs.StringVar(&configPath, "config", configPath, "path to the JSON configuration file")

	if err := fs.Parse(args); err != nil {
		return "", err
	}

	fs.Visit(func(f *flag.Flag) {
		if f.Name == "i" {
			cfg.StoreInterval = time.Duration(storeIntervalSeconds) * time.Second
		}
	})

	return configPath, nil
}

func loadServerFile(path string, cfg *ServerConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read server config file %q: %w", path, err)
	}

	var fileCfg serverFileConfig
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("parse server config file %q: %w", path, err)
	}

	if fileCfg.Address != nil {
		cfg.Address = *fileCfg.Address
	}
	if fileCfg.GRPCAddress != nil {
		cfg.GRPCAddress = *fileCfg.GRPCAddress
	}
	if fileCfg.GRPCCertFile != nil {
		cfg.GRPCCertFile = *fileCfg.GRPCCertFile
	}
	if fileCfg.GRPCKeyFile != nil {
		cfg.GRPCKeyFile = *fileCfg.GRPCKeyFile
	}
	if fileCfg.Restore != nil {
		cfg.Restore = *fileCfg.Restore
	}
	if fileCfg.StoreInterval != nil {
		cfg.StoreInterval, err = time.ParseDuration(*fileCfg.StoreInterval)
		if err != nil {
			return fmt.Errorf("invalid store_interval in server config file %q: %w", path, err)
		}
	}
	if fileCfg.StoreFile != nil {
		cfg.FileStoragePath = *fileCfg.StoreFile
	}
	if fileCfg.DatabaseDSN != nil {
		cfg.DatabaseDSN = *fileCfg.DatabaseDSN
	}
	if fileCfg.Key != nil {
		cfg.Key = *fileCfg.Key
	}
	if fileCfg.CryptoKey != nil {
		cfg.CryptoKey = *fileCfg.CryptoKey
	}
	if fileCfg.AuditFile != nil {
		cfg.AuditFile = *fileCfg.AuditFile
	}
	if fileCfg.AuditURL != nil {
		cfg.AuditURL = *fileCfg.AuditURL
	}
	if fileCfg.TrustedSubnet != nil {
		cfg.TrustedSubnet = *fileCfg.TrustedSubnet
	}

	return nil
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
