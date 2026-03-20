package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/transfans/payment/internal/db"
	"github.com/transfans/payment/internal/httputil"
	"github.com/transfans/payment/internal/middleware"
)

type createPayoutRequest struct {
	Amount float64 `json:"amount"`
}

type createPayoutResponse struct {
	PayoutID    string  `json:"payout_id"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	CompletedAt string  `json:"completed_at"`
}

type payoutItem struct {
	ID          string  `json:"id"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	CompletedAt string  `json:"completed_at"`
}

type payoutsResponse struct {
	Items []payoutItem `json:"items"`
	Total int64        `json:"total"`
}

func (a *App) CreatePayout(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r.Context())

	var req createPayoutRequest
	if err := httputil.ReadJSON(w, r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Amount <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	var creatorID pgtype.UUID
	if err := creatorID.Scan(claims.UserID); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var amount pgtype.Numeric
	if err := amount.Scan(strconv.FormatFloat(req.Amount, 'f', 2, 64)); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid amount")
		return
	}

	_, err := a.Queries.DecrementBalance(r.Context(), db.DecrementBalanceParams{
		CreatorID: creatorID,
		Available: amount,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.WriteError(w, http.StatusBadRequest, "amount exceeds available balance")
			return
		}
		a.Logger.Error("decrement balance", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	payout, err := a.Queries.InsertPayout(r.Context(), db.InsertPayoutParams{
		CreatorID: creatorID,
		Amount:    amount,
	})
	if err != nil {
		a.Logger.Error("insert payout", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, createPayoutResponse{
		PayoutID:    uuidToString(payout.ID),
		Amount:      numericToFloat64(payout.Amount),
		Status:      payout.Status,
		CompletedAt: timeToString(payout.CompletedAt),
	})
}

func (a *App) ListPayouts(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r.Context())

	var creatorID pgtype.UUID
	if err := creatorID.Scan(claims.UserID); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	limit, offset := httputil.ParsePage(r, 20)

	total, err := a.Queries.CountPayoutsByCreator(r.Context(), creatorID)
	if err != nil {
		a.Logger.Error("count payouts", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	rows, err := a.Queries.ListPayoutsByCreator(r.Context(), db.ListPayoutsByCreatorParams{
		CreatorID: creatorID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		a.Logger.Error("list payouts", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	items := make([]payoutItem, len(rows))
	for i, row := range rows {
		items[i] = payoutItem{
			ID:          uuidToString(row.ID),
			Amount:      numericToFloat64(row.Amount),
			Status:      row.Status,
			CompletedAt: timeToString(row.CompletedAt),
		}
	}

	httputil.WriteJSON(w, http.StatusOK, payoutsResponse{Items: items, Total: total})
}
