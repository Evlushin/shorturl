package main

import (
	"context"
	"github.com/Evlushin/shorturl/internal/logger"
	"log"
	"os"
	"os/signal"

	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/handler"
	"github.com/Evlushin/shorturl/internal/service"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	cfg := config.GetConfig()

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	shortenerService, err := service.NewShortener(cfg)
	if err != nil {
		return err
	}
	defer shortenerService.Close()

	handler.Serve(ctx, cfg.Handlers, shortenerService)
	return nil
}
