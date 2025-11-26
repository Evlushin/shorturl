package handler

import (
	"context"
	"errors"
	"github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/middleware"
	"github.com/Evlushin/shorturl/internal/observers"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

const defaultShutdownCtxTimeout = 10 * time.Second

func Serve(
	ctx context.Context,
	cfg config.Config,
	shortener Shortener,
) {
	auditManager := observers.InitAuditObservers(ctx, cfg)

	h := newHandlers(shortener, cfg, auditManager)
	router := newRouter(h)

	httpServer := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	var wg sync.WaitGroup

	// Горутина для запуска сервера
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Log.Info("starting server", zap.String("addr", cfg.ServerAddr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("error starting server", zap.Error(err))
		}
	}()

	// Горутина для обработки завершения
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done() // Ждём отмены контекста

		// Используем родительский контекст для таймаута
		shutdownCtx, cancel := context.WithTimeout(ctx, defaultShutdownCtxTimeout)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Log.Error("error shutting down http server", zap.Error(err))
		}
		logger.Log.Info("close server", zap.String("addr", cfg.ServerAddr))

		auditManager.Stop()
		logger.Log.Info("stop audit manager")
	}()

	// Ждём завершения обеих горутин
	wg.Wait()
}

func newRouter(h *handlers) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logger.RequestLogger)
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.UserIDMiddleware(&h.cfg))

	r.Post("/", h.SetShortener)
	r.Get("/{id}", h.GetShortener)
	r.Get("/ping", h.Ping)

	r.Route("/api", func(r chi.Router) {
		r.Route("/shorten", func(r chi.Router) {
			r.Post("/", h.SetShortenerAPI)
			r.Post("/batch", h.SetShortenerBatchAPI)
		})
		r.Route("/user", func(r chi.Router) {
			r.Route("/urls", func(r chi.Router) {
				r.Get("/", h.GetShortenerUrlsAPI)
				r.Delete("/", h.DeleteShortenerUrlsAPI)
			})
		})
	})

	return r
}
