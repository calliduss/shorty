package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"shorty/internal/config"
	mwLogger "shorty/internal/server/middleware/logger"
	"shorty/internal/storage/sqlite"
)

type router struct {
	storage sqlite.Storage
	log     *slog.Logger
}

func SetupRouter(st *sqlite.Storage, cfg config.Config, log *slog.Logger) http.Handler {
	ro := &router{
		storage: *st,
		log:     log,
	}

	r := chi.NewRouter()
	r.Use(
		middleware.RequestID,
		middleware.Logger,
		mwLogger.New(log),
		middleware.Recoverer,
		middleware.URLFormat,
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

	r.Get("/{alias}", ro.getURLHandler)
	r.Route("/url", func(r chi.Router) {
		r.Post("/", ro.saveAliasHandler)
		r.Delete("/delete_user_alias", ro.deleteAliasHandler)
	})
}
