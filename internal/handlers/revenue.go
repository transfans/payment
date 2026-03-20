package handlers

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/transfans/payment/internal/db"
	"github.com/transfans/payment/internal/httputil"
	"github.com/transfans/payment/internal/middleware"
)

type revenueByTierItem struct {
	TierID string  `json:"tier_id"`
	Count  int32   `json:"count"`
	Amount float64 `json:"amount"`
}

type revenueResponse struct {
	Total  float64             `json:"total"`
	ByTier []revenueByTierItem `json:"by_tier"`
}

func (a *App) GetRevenue(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r.Context())

	var creatorID pgtype.UUID
	if err := creatorID.Scan(claims.UserID); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	from, to, err := parseDateRange(r)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid date format, use ISO 8601")
		return
	}

	type totalResult struct {
		val pgtype.Numeric
		err error
	}
	type rowsResult struct {
		val []db.GetRevenueByTierRow
		err error
	}

	totalCh := make(chan totalResult, 1)
	rowsCh := make(chan rowsResult, 1)

	go func() {
		val, err := a.Queries.GetRevenue(r.Context(), db.GetRevenueParams{
			CreatorID: creatorID,
			Column2:   from,
			Column3:   to,
		})
		totalCh <- totalResult{val, err}
	}()

	go func() {
		val, err := a.Queries.GetRevenueByTier(r.Context(), db.GetRevenueByTierParams{
			CreatorID: creatorID,
			Column2:   from,
			Column3:   to,
		})
		rowsCh <- rowsResult{val, err}
	}()

	tr := <-totalCh
	rr := <-rowsCh

	if tr.err != nil {
		a.Logger.Error("get revenue", "error", tr.err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if rr.err != nil {
		a.Logger.Error("get revenue by tier", "error", rr.err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	total := tr.val
	rows := rr.val

	byTier := make([]revenueByTierItem, len(rows))
	for i, row := range rows {
		byTier[i] = revenueByTierItem{
			TierID: uuidToString(row.TierID),
			Count:  row.Count,
			Amount: numericToFloat64(row.Amount),
		}
	}

	httputil.WriteJSON(w, http.StatusOK, revenueResponse{
		Total:  numericToFloat64(total),
		ByTier: byTier,
	})
}

func parseDateRange(r *http.Request) (from, to pgtype.Timestamptz, err error) {
	if s := r.URL.Query().Get("from"); s != "" {
		t, parseErr := time.Parse(time.RFC3339, s)
		if parseErr != nil {
			return from, to, parseErr
		}
		from = pgtype.Timestamptz{Time: t, Valid: true}
	}
	if s := r.URL.Query().Get("to"); s != "" {
		t, parseErr := time.Parse(time.RFC3339, s)
		if parseErr != nil {
			return from, to, parseErr
		}
		to = pgtype.Timestamptz{Time: t, Valid: true}
	}
	return from, to, nil
}
