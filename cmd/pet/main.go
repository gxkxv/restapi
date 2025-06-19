package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"os"

	"github.com/gxkxv/restapi-pet/internal/lib/logger/sl"
	"github.com/gxkxv/restapi-pet/internal/storage/postgresql"

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

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Route("/", func(r chi.Router) {
		r.Get("/users", postgresql.GetUsers(storage))
		r.Get("/info", postgresql.GetUser(storage))
		r.Get("/create", postgresql.CreateUser(storage))
		r.Post("/create", postgresql.CreateUser(storage))
		r.Patch("/update/{name}", postgresql.UpdateUser(storage))

	})
	err = http.ListenAndServe(":8080", router)
	if err != nil {
		log.Error("error starting server", sl.Err(err))
	}
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
