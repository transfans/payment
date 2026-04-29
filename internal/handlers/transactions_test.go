package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/transfans/payment/internal/db"
)

func TestListTransactions_Empty(t *testing.T) {
	q := &mockQuerier{
		countTransactions: func(_ context.Context, _ pgtype.UUID) (int64, error) { return 0, nil },
		listTransactions:  func(_ context.Context, _ db.ListTransactionsByFanParams) ([]db.Transaction, error) {
			return nil, nil
		},
	}
	app := newApp(q, nil, nil, nil)
	h := applyAuth(app, app.ListTransactions)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authReq(http.MethodGet, "/transactions", "", fanID, "fan"))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", rec.Code)
	}
	var resp struct {
		Items []any `json:"items"`
		Total int64 `json:"total"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Total != 0 {
		t.Errorf("want 0 total got %d", resp.Total)
	}
}

func TestListTransactions_WithItems(t *testing.T) {
	var id pgtype.UUID
	_ = id.Scan(txID)
	var cid pgtype.UUID
	_ = cid.Scan(creatorID)
	var tid pgtype.UUID
	_ = tid.Scan(tierID)
	var amount pgtype.Numeric
	_ = amount.Scan("9.99")
	ts := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	q := &mockQuerier{
		countTransactions: func(_ context.Context, _ pgtype.UUID) (int64, error) { return 1, nil },
		listTransactions: func(_ context.Context, _ db.ListTransactionsByFanParams) ([]db.Transaction, error) {
			return []db.Transaction{{
				ID: id, CreatorID: cid, TierID: tid,
				Amount: amount, Status: "success", CreatedAt: ts,
			}}, nil
		},
	}
	app := newApp(q, nil, nil, nil)
	h := applyAuth(app, app.ListTransactions)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authReq(http.MethodGet, "/transactions", "", fanID, "fan"))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", rec.Code)
	}
	var resp struct {
		Items []struct {
			ID     string  `json:"id"`
			Amount float64 `json:"amount"`
			Status string  `json:"status"`
		} `json:"items"`
		Total int64 `json:"total"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Errorf("want 1 got %d", resp.Total)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("want 1 item got %d", len(resp.Items))
	}
	if resp.Items[0].Status != "success" {
		t.Errorf("want success got %s", resp.Items[0].Status)
	}
}
