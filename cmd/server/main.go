package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/transfans/payment/apidocs"
	"github.com/transfans/payment/internal/config"
	"github.com/transfans/payment/internal/db"
	"github.com/transfans/payment/internal/handlers"
	"github.com/transfans/payment/internal/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	if err := db.Migrate(sqlDB); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	sqlDB.Close()
	logger.Info("migrations applied")

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	app := &handlers.App{
		Queries: db.New(pool),
		Logger:  logger,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.SharedJWTSecret))

		r.Post("/checkout", func(w http.ResponseWriter, r *http.Request) {})
		r.Get("/transactions", app.ListTransactions)

		r.Group(func(r chi.Router) {
			r.Use(middleware.CreatorOnly)

			r.Get("/balance", app.GetBalance)
			r.Post("/payouts", app.CreatePayout)
			r.Get("/payouts", app.ListPayouts)
			r.Get("/revenue", app.GetRevenue)
		})
	})

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(apidocs.SwaggerHTML)
	})
	r.Get("/docs/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(apidocs.OpenAPISpec)
	})

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
