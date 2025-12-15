package config

import (
	"github.com/ilyakaznacheev/cleanenv"

	handlersConfig "github.com/Evlushin/shorturl/internal/handler/config"
)

type Config struct {
	Handlers      handlersConfig.Config
	LogLevel      string `env:"LOG_LEVEL" flag:"l" env-default:"info" env-description:"log level"`
	FileStorePath string `env:"FILE_STORAGE_PATH" flag:"f" env-default:"" env-description:"address storage"`
	DatabaseDsn   string `env:"DATABASE_DSN" flag:"d" env-default:"" env-description:"connection string"`
}

func GetConfig() (Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
