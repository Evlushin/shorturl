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
	DatabaseDsn   string
}

func GetConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.Handlers.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.Handlers.BaseAddr, "b", "http://localhost:8080", "base address of the resulting shortened URL")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	//flag.StringVar(&cfg.FileStorePath, "f", "storage.txt", "address storage")
	flag.StringVar(&cfg.DatabaseDsn, "d", "host=127.127.126.41 port=5432 dbname=shorturl user=shorturl password=shorturl connect_timeout=10 sslmode=prefer", "connection string")
	flag.StringVar(&cfg.FileStorePath, "f", "", "address storage")
	//flag.StringVar(&cfg.DatabaseDsn, "d", "", "connection string")
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

	if databaseDsn := os.Getenv("DATABASE_DSN"); databaseDsn != "" {
		cfg.DatabaseDsn = databaseDsn
	}

	return cfg
}
