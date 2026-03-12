package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/transfans/payment/internal/config"
	"github.com/transfans/payment/internal/db"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	conn, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err := db.Migrate(conn); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")

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
