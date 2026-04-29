package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/transfans/payment/internal/db"
)

func TestCreatePayout_InvalidAmount(t *testing.T) {
	app := newApp(&mockQuerier{}, nil, nil, nil)
	h := applyAuth(app, app.CreatePayout)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authReq(http.MethodPost, "/payouts", `{"amount":0}`, creatorID, "creator"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d", rec.Code)
	}
}

func TestCreatePayout_InsufficientBalance(t *testing.T) {
	q := &mockQuerier{
		decrementBalance: func(_ context.Context, _ db.DecrementBalanceParams) (db.Balance, error) {
			return db.Balance{}, pgx.ErrNoRows
		},
	}
	app := newApp(q, nil, nil, nil)
	h := applyAuth(app, app.CreatePayout)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authReq(http.MethodPost, "/payouts", `{"amount":999.99}`, creatorID, "creator"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreatePayout_Success(t *testing.T) {
	var pid pgtype.UUID
	_ = pid.Scan("00000000-0000-0000-0000-000000000005")
	var amount pgtype.Numeric
	_ = amount.Scan("50.00")
	ts := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	q := &mockQuerier{
		decrementBalance: func(_ context.Context, _ db.DecrementBalanceParams) (db.Balance, error) {
			return db.Balance{}, nil
		},
		insertPayout: func(_ context.Context, _ db.InsertPayoutParams) (db.Payout, error) {
			return db.Payout{ID: pid, Amount: amount, Status: "completed", CompletedAt: ts}, nil
		},
	}
	app := newApp(q, nil, nil, nil)
	h := applyAuth(app, app.CreatePayout)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authReq(http.MethodPost, "/payouts", `{"amount":50.00}`, creatorID, "creator"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("want 201 got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Status string `json:"status"`
		Amount float64 `json:"amount"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "completed" {
		t.Errorf("want completed got %s", resp.Status)
	}
}

func TestListPayouts_Empty(t *testing.T) {
	q := &mockQuerier{
		countPayouts: func(_ context.Context, _ pgtype.UUID) (int64, error) { return 0, nil },
		listPayouts:  func(_ context.Context, _ db.ListPayoutsByCreatorParams) ([]db.Payout, error) { return nil, nil },
	}
	app := newApp(q, nil, nil, nil)
	h := applyAuth(app, app.ListPayouts)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authReq(http.MethodGet, "/payouts", "", creatorID, "creator"))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", rec.Code)
	}
	var resp struct {
		Total int64 `json:"total"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Total != 0 {
		t.Errorf("want 0 got %d", resp.Total)
	}
}
