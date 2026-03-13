package handlers

import (
	"log/slog"

	"github.com/transfans/payment/internal/db"
)

type App struct {
	Queries *db.Queries
	Logger  *slog.Logger
}
