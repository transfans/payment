package handlers

import (
	"log/slog"

	"github.com/transfans/payment/internal/db"
	"github.com/transfans/payment/internal/mq"
	"github.com/transfans/payment/internal/profile"
)

type App struct {
	Queries       *db.Queries
	Logger        *slog.Logger
	Publisher     *mq.Publisher
	ProfileClient *profile.Client
}
