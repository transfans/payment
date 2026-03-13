package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/transfans/payment/internal/db"
	"github.com/transfans/payment/internal/httputil"
	"github.com/transfans/payment/internal/middleware"
)

type transactionItem struct {
	ID        string  `json:"id"`
	CreatorID string  `json:"creator_id"`
	TierID    string  `json:"tier_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

type transactionsResponse struct {
	Items []transactionItem `json:"items"`
	Total int64             `json:"total"`
}

func (a *App) ListTransactions(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r.Context())

	var fanID pgtype.UUID
	if err := fanID.Scan(claims.UserID); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	limit, offset := httputil.ParsePage(r, 20)

	total, err := a.Queries.CountTransactionsByFan(r.Context(), fanID)
	if err != nil {
		a.Logger.Error("count transactions", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	rows, err := a.Queries.ListTransactionsByFan(r.Context(), db.ListTransactionsByFanParams{
		FanID:  fanID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		a.Logger.Error("list transactions", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	items := make([]transactionItem, len(rows))
	for i, row := range rows {
		items[i] = transactionItem{
			ID:        uuidToString(row.ID),
			CreatorID: uuidToString(row.CreatorID),
			TierID:    uuidToString(row.TierID),
			Amount:    numericToFloat64(row.Amount),
			Status:    row.Status,
			CreatedAt: timeToString(row.CreatedAt),
		}
	}

	httputil.WriteJSON(w, http.StatusOK, transactionsResponse{Items: items, Total: total})
}
