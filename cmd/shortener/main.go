package main

import (
	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/repository"
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

	store, err := repository.NewRepository(&cfg)
	if err != nil {
		return err
	}
	defer store.Close()

	shortenerService := service.NewShortener(store)

	return handler.Serve(cfg.Handlers, shortenerService)
}
