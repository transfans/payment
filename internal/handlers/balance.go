package handlers

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/transfans/payment/internal/httputil"
	"github.com/transfans/payment/internal/middleware"
)

type balanceResponse struct {
	Available   float64 `json:"available"`
	TotalEarned float64 `json:"total_earned"`
	TotalPaid   float64 `json:"total_paid"`
}

func (a *App) GetBalance(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r.Context())

	var creatorID pgtype.UUID
	if err := creatorID.Scan(claims.UserID); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	bal, err := a.Queries.GetBalance(r.Context(), creatorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.WriteJSON(w, http.StatusOK, balanceResponse{})
			return
		}
		a.Logger.Error("get balance", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, balanceResponse{
		Available:   numericToFloat64(bal.Available),
		TotalEarned: numericToFloat64(bal.TotalEarned),
		TotalPaid:   numericToFloat64(bal.TotalPaid),
	})
}
