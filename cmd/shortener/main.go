package main

import (
	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/repository"
	"github.com/Evlushin/shorturl/internal/repository/file"
	"github.com/Evlushin/shorturl/internal/repository/inmemory"
	"github.com/Evlushin/shorturl/internal/repository/pg"
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

	store, err := NewRepository(&cfg)
	defer store.Close()

	if err != nil {
		return err
	}

	shortenerService := service.NewShortener(store)

	return handler.Serve(cfg.Handlers, shortenerService)
}

func NewRepository(cfg *config.Config) (repository.Repository, error) {
	if cfg.DatabaseDsn != "" {
		return pg.NewStore(cfg)
	}

	if cfg.FileStorePath != "" {
		return file.NewStore(cfg)
	}

	return inmemory.NewStore(cfg)
}
