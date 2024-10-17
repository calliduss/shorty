package main

import (
	"log/slog"
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
	//todo: run server
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
