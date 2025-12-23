package config

import (
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"

	handlersConfig "github.com/Evlushin/shorturl/internal/handler/config"
)

type Config struct {
	Handlers      handlersConfig.Config
	LogLevel      string `env:"LOG_LEVEL" env-default:"info" env-description:"log level"`
	FileStorePath string `env:"FILE_STORAGE_PATH" env-default:"" env-description:"address storage"`
	DatabaseDsn   string `env:"DATABASE_DSN" env-default:"" env-description:"connection string"`
	ConfigFile    string `env:"CONFIG" env-description:"config file"`
}

func GetConfig() (Config, error) {
	var cfg Config

	flag.StringVar(&cfg.Handlers.Audit.AuditFile, "audit-file", "audit.txt", "path to the destination file where audit logs are saved")
	flag.StringVar(&cfg.Handlers.Audit.AuditURL, "audit-url", "", "the full URL of the remote receiving server where the audit logs are sent")
	flag.StringVar(&cfg.Handlers.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.Handlers.BaseAddr, "b", "http://localhost:8080", "base address of the resulting shortened URL")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	//flag.StringVar(&cfg.FileStorePath, "f", "storage.txt", "address storage")
	//flag.StringVar(&cfg.DatabaseDsn, "d", "host=127.127.126.41 port=5432 dbname=shorturl user=shorturl password=shorturl connect_timeout=10 sslmode=prefer", "connection string")
	flag.StringVar(&cfg.FileStorePath, "f", "", "address storage")
	flag.StringVar(&cfg.DatabaseDsn, "d", "", "connection string")
	flag.StringVar(&cfg.Handlers.SecretKey, "k", uuid.NewString(), "secret key")
	flag.BoolVar(&cfg.Handlers.EnableHTTPS, "s", false, "enable https")
	flag.StringVar(&cfg.ConfigFile, "c", ".env", "config file")
	flag.Parse()
	if err := cleanenv.ReadConfig(cfg.ConfigFile, &cfg); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	return cfg, nil
}
