package handlers

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/transfans/payment/internal/db"
	"github.com/transfans/payment/internal/mq"
	"github.com/transfans/payment/internal/profile"
)

type App struct {
	Pool          *pgxpool.Pool
	Queries       *db.Queries
	Logger        *slog.Logger
	Publisher     *mq.Publisher
	ProfileClient *profile.Client
}
