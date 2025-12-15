package config

import (
	"github.com/ilyakaznacheev/cleanenv"

	handlersConfig "github.com/Evlushin/shorturl/internal/handler/config"
)

type Config struct {
	Handlers      handlersConfig.Config
	LogLevel      string `env:"LOG_LEVEL" flag:"l" default:"info" help:"log level"`
	FileStorePath string `env:"FILE_STORAGE_PATH" flag:"f" default:"" help:"address storage"`
	DatabaseDsn   string `env:"DATABASE_DSN" flag:"d" default:"" help:"connection string"`
}

func GetConfig() (Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
