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
	"net"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"
)

const defaultShutdownCtxTimeout = 10 * time.Second

func Serve(ctx context.Context, cfg config.Config, shortener Shortener) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	auditManager := observers.InitAuditObservers(ctx, cfg)
	h := newHandlers(shortener, cfg, auditManager)
	router := newRouter(h)

	httpServer := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	var wg sync.WaitGroup

	// Запуск сервера
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Log.Info("starting server", zap.String("addr", cfg.ServerAddr))
		lis, err := net.Listen("tcp", cfg.ServerAddr)
		if err != nil {
			logger.Log.Error("failed to listen", zap.Error(err))
			cancel()
			return
		}
		if err := httpServer.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("error serving", zap.Error(err))
			cancel()
		}
	}()

	// Завершение
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, defaultShutdownCtxTimeout)
		defer shutdownCancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Log.Error("error shutting down http server", zap.Error(err))
		}

		auditManager.Stop()

		logger.Log.Info("server shutdown complete")
	}()

	wg.Wait()
}

func newRouter(h *handlers) *chi.Mux {
	r := chi.NewRouter()

	// pprof
	r.Route("/debug/pprof", func(r chi.Router) {
		r.Handle("/", http.HandlerFunc(pprof.Index))
		r.Handle("/profile", http.HandlerFunc(pprof.Profile))
		r.Handle("/symbol", http.HandlerFunc(pprof.Symbol))
		r.Handle("/cmdline", http.HandlerFunc(pprof.Cmdline))
		r.Handle("/heap", pprof.Handler("heap"))
	})
	r.Group(func(r chi.Router) {
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
	})

	return r
}
