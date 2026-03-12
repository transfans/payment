package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/transfans/payment/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	r.Post("/checkout", func(w http.ResponseWriter, r *http.Request) {})
	r.Get("/transactions", func(w http.ResponseWriter, r *http.Request) {})
	r.Get("/balance", func(w http.ResponseWriter, r *http.Request) {})
	r.Post("/payouts", func(w http.ResponseWriter, r *http.Request) {})
	r.Get("/payouts", func(w http.ResponseWriter, r *http.Request) {})
	r.Get("/revenue", func(w http.ResponseWriter, r *http.Request) {})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("starting server", "port", cfg.Port)

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
