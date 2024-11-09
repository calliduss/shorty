package main

import (
	"log/slog"
	"net/http"
	"os"
	"shorty/internal/config"
	"shorty/internal/pkg/logger/slo"
	"shorty/internal/server"
	"shorty/internal/storage/sqlite"
)

const (
	envLocal = "local"
	envProd  = "prod"
)

func main() {
	cfg := config.InitConfig()
	log := setupLogger(cfg.Environment)
	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to init storage", slo.Err(err))
		os.Exit(1)
	}

	router := server.SetupRouter(storage, *cfg, log)
	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	//TODO: add graceful shutdown
	srv := startHTTPServer(cfg, router)

	err = srv.ListenAndServe()
	if err != nil {
		log.Error("failed to start server", slog.String("address", cfg.HTTPServer.Address))
	}

	log.Error("server stopped", slog.String("address", cfg.HTTPServer.Address))
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return log
}

func startHTTPServer(cfg *config.Config, router http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}
}
