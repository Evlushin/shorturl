package main

import (
	"log"

	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/handler"
	"github.com/Evlushin/shorturl/internal/repository"
	"github.com/Evlushin/shorturl/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.GetConfig()

	store := repository.NewStore()
	shortenerService := service.NewShortener(store)

	return handler.Serve(cfg.Handlers, shortenerService)
}
