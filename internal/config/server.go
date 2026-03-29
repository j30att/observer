package config

type ServerConfig struct {
	Address string
}

func NewServerConfig() ServerConfig {
	return ServerConfig{
		Address: "localhost:8080",
	}
}
