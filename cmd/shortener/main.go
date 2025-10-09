package main

import (
	"github.com/Evlushin/shorturl/internal/logger"
	"log"

	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/handler"
	"github.com/Evlushin/shorturl/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.GetConfig()

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	shortenerService, err := service.NewShortener(cfg)
	if err != nil {
		return err
	}
	defer shortenerService.Close()

	return handler.Serve(cfg.Handlers, shortenerService)
}
