package config

import (
	"flag"
	handlersConfig "github.com/Evlushin/shorturl/internal/handler/config"
)

type Config struct {
	Handlers handlersConfig.Config
}

func GetConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.Handlers.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.Handlers.BaseAddr, "b", "localhost:8080", "base address of the resulting shortened URL")

	flag.Parse()
	return cfg
}
