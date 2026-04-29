package handlers

import (
	"log/slog"

	"github.com/transfans/payment/internal/db"
)

type App struct {
	Pool          TxBeginner
	Queries       db.Querier
	Logger        *slog.Logger
	Publisher     MQPublisher
	ProfileClient ProfileClient
}
