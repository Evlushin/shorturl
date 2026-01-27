package handler

import (
	"net/http"
	"net/http/pprof"

	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/middleware"
	"github.com/go-chi/chi/v5"
)

func newRouter(h *Handlers) *chi.Mux {
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
			r.Route("/internal", func(r chi.Router) {
				r.Use(middleware.TrustedSubnetMiddleware(h.cfg.TrustedSubnet))
				r.Get("/stats", h.GetStats)
			})
		})
	})

	return r
}
