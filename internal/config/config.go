package config

import (
	"flag"
	handlersConfig "github.com/Evlushin/shorturl/internal/handler/config"
	"os"
)

type Config struct {
	Handlers      handlersConfig.Config
	LogLevel      string
	FileStorePath string
}

func GetConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.Handlers.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.Handlers.BaseAddr, "b", "http://localhost:8080", "base address of the resulting shortened URL")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.StringVar(&cfg.FileStorePath, "f", "storage.txt", "address storage")
	flag.StringVar(&cfg.FileStorePath, "file-storage-path", "storage.txt", "address storage")
	flag.Parse()

	if serverAddr := os.Getenv("SERVER_ADDRESS"); serverAddr != "" {
		cfg.Handlers.ServerAddr = serverAddr
	}

	if baseAddr := os.Getenv("BASE_URL"); baseAddr != "" {
		cfg.Handlers.BaseAddr = baseAddr
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	if fileStorePath := os.Getenv("FILE_STORAGE_PATH"); fileStorePath != "" {
		cfg.FileStorePath = fileStorePath
	}

	return cfg
}
