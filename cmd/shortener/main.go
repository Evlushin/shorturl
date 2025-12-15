package main

import (
	"context"
	"fmt"
	"github.com/Evlushin/shorturl/internal/utils"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"

	"github.com/Evlushin/shorturl/internal/logger"

	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/handler"
	"github.com/Evlushin/shorturl/internal/service"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
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

	printBuildInfo()

	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("get config: %w", err)
	}

	if err = logger.Initialize(cfg.LogLevel); err != nil {
		return fmt.Errorf("loger init: %w", err)
	}

	shortenerService, err := service.NewShortener(cfg)
	if err != nil {
		return fmt.Errorf("service init: %w", err)
	}
	defer func(shortenerService *service.Shortener) {
		err = shortenerService.Close()
		if err != nil {
			logger.Log.Error("service close", zap.Error(err))
			return
		}
	}(shortenerService)

	handler.Serve(ctx, cfg.Handlers, shortenerService)
	return nil
}

// printBuildInfo выводит информацию о сборке
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", utils.FormatValue(buildVersion))
	fmt.Printf("Build date: %s\n", utils.FormatValue(buildDate))
	fmt.Printf("Build commit: %s\n", utils.FormatValue(buildCommit))
}
