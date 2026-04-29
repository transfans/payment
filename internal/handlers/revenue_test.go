package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/transfans/payment/internal/db"
)

func TestGetRevenue_InvalidDateFormat(t *testing.T) {
	q := &mockQuerier{
		getRevenue:       func(_ context.Context, _ db.GetRevenueParams) (pgtype.Numeric, error) { return pgtype.Numeric{}, nil },
		getRevenueByTier: func(_ context.Context, _ db.GetRevenueByTierParams) ([]db.GetRevenueByTierRow, error) { return nil, nil },
	}
	app := newApp(q, nil, nil, nil)
	h := applyAuth(app, app.GetRevenue)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authReq(http.MethodGet, "/revenue?from=not-a-date", "", creatorID, "creator"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d", rec.Code)
	}
}

func TestGetRevenue_NoDateFilter(t *testing.T) {
	var total pgtype.Numeric
	_ = total.Scan("300.00")

	var tid pgtype.UUID
	_ = tid.Scan(tierID)
	var tierAmount pgtype.Numeric
	_ = tierAmount.Scan("300.00")

	q := &mockQuerier{
		getRevenue: func(_ context.Context, _ db.GetRevenueParams) (pgtype.Numeric, error) {
			return total, nil
		},
		getRevenueByTier: func(_ context.Context, _ db.GetRevenueByTierParams) ([]db.GetRevenueByTierRow, error) {
			return []db.GetRevenueByTierRow{{TierID: tid, Count: 3, Amount: tierAmount}}, nil
		},
	}
	app := newApp(q, nil, nil, nil)
	h := applyAuth(app, app.GetRevenue)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authReq(http.MethodGet, "/revenue", "", creatorID, "creator"))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Total  float64 `json:"total"`
		ByTier []struct {
			TierID string  `json:"tier_id"`
			Count  int32   `json:"count"`
			Amount float64 `json:"amount"`
		} `json:"by_tier"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Total != 300 {
		t.Errorf("want 300 got %v", resp.Total)
	}
	if len(resp.ByTier) != 1 {
		t.Fatalf("want 1 by_tier got %d", len(resp.ByTier))
	}
	if resp.ByTier[0].Count != 3 {
		t.Errorf("want 3 got %d", resp.ByTier[0].Count)
	}
}
