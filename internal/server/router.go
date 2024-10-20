package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"shorty/internal/config"
	mwLogger "shorty/internal/server/middleware/logger"
)

type router struct {
	storage UrlProvider
	log     *slog.Logger
}

func SetupRouter(storage UrlProvider, cfg config.Config, log *slog.Logger) http.Handler {
	ro := &router{
		storage: storage,
		log:     log,
	}

	r := chi.NewRouter()
	r.Use(
		middleware.RequestID,
		middleware.Logger,
		mwLogger.New(log),
		middleware.Recoverer,
		middleware.URLFormat, // /{alias}
	)

	r.Route("/v1", func(r chi.Router) {
		ro.registerHandlers(r, cfg)
	})

	return r
}

func (ro *router) registerHandlers(r chi.Router, cfg config.Config) {
	r.Use(middleware.BasicAuth("shorty", map[string]string{
		cfg.HTTPServer.User: cfg.HTTPServer.Password,
	}))

	r.Get("/{alias}", ro.redirectHandler)
	r.Route("/url", func(r chi.Router) {
		r.Post("/", ro.saveAliasHandler)
		r.Delete("/{alias}", ro.deleteAliasHandler)
		r.Patch("/{alias}", ro.updateAliasHandler)
	})
}
