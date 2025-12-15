package main

import (
	"context"
	"fmt"
	"github.com/Evlushin/shorturl/internal/utils"
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
	// Выводим информацию о сборке при старте
	printBuildInfo()

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

// printBuildInfo выводит информацию о сборке
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", utils.FormatValue(buildVersion))
	fmt.Printf("Build date: %s\n", utils.FormatValue(buildDate))
	fmt.Printf("Build commit: %s\n", utils.FormatValue(buildCommit))
}
