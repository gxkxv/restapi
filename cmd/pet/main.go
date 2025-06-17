package main

import (
	"github.com/gxkxv/restapi-pet/internal/lib/logger/sl"
	"github.com/gxkxv/restapi-pet/internal/storage/postgresql"
	"log/slog"
	"os"

	"github.com/gxkxv/restapi-pet/internal/config"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("start", "env", cfg.Env)
	log.Debug("debug messages are enabled")

	storage, err := postgresql.New(cfg)
	if err != nil {
		log.Error("error creating postgresql storage", sl.Err(err))
		os.Exit(1)
	}

	_ = storage
	storage.GetInfoFromURL("timur")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return log
}
