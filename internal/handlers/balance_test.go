package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/transfans/payment/internal/db"
)

func TestGetBalance_NoRows(t *testing.T) {
	q := &mockQuerier{
		getBalance: func(_ context.Context, _ pgtype.UUID) (db.Balance, error) {
			return db.Balance{}, pgx.ErrNoRows
		},
	}
	app := newApp(q, nil, nil, nil)
	h := applyAuth(app, app.GetBalance)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authReq(http.MethodGet, "/balance", "", creatorID, "creator"))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", rec.Code)
	}
	var resp map[string]float64
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["available"] != 0 || resp["total_earned"] != 0 {
		t.Errorf("expected zero balance, got %v", resp)
	}
}

func TestGetBalance_Found(t *testing.T) {
	var avail, earned, paid pgtype.Numeric
	_ = avail.Scan("100.00")
	_ = earned.Scan("200.00")
	_ = paid.Scan("100.00")

	q := &mockQuerier{
		getBalance: func(_ context.Context, _ pgtype.UUID) (db.Balance, error) {
			return db.Balance{Available: avail, TotalEarned: earned, TotalPaid: paid}, nil
		},
	}
	app := newApp(q, nil, nil, nil)
	h := applyAuth(app, app.GetBalance)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authReq(http.MethodGet, "/balance", "", creatorID, "creator"))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Available   float64 `json:"available"`
		TotalEarned float64 `json:"total_earned"`
		TotalPaid   float64 `json:"total_paid"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Available != 100 {
		t.Errorf("want 100 got %v", resp.Available)
	}
	if resp.TotalEarned != 200 {
		t.Errorf("want 200 got %v", resp.TotalEarned)
	}
}
