package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/transfans/payment/internal/db"
	"github.com/transfans/payment/internal/handlers"
	"github.com/transfans/payment/internal/middleware"
)

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping integration tests")
	}

	// Run migrations via database/sql + goose so tables exist.
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(sqlDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB.Close()

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Clean up between tests.
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM payouts")
		pool.Exec(context.Background(), "DELETE FROM balances")
		pool.Exec(context.Background(), "DELETE FROM transactions")
	})

	return pool
}

func integrationApp(t *testing.T, pool *pgxpool.Pool) *handlers.App {
	t.Helper()
	pub := &mockPublisher{
		publish: func(_ context.Context, _, _ string, _ any) error { return nil },
	}
	pc := successProfileClient()
	return &handlers.App{
		Pool:          pool,
		Queries:       db.NewQuerier(pool),
		Logger:        newApp(nil, nil, nil, nil).Logger,
		Publisher:     pub,
		ProfileClient: pc,
	}
}

func applyAuthIntegration(next http.HandlerFunc) http.Handler {
	return middleware.Auth(testJWTSecret)(next)
}

func TestIntegration_CheckoutAndBalance(t *testing.T) {
	pool := integrationPool(t)
	app := integrationApp(t, pool)

	// Checkout
	h := applyAuthIntegration(app.Checkout)
	req := authReq(http.MethodPost, "/checkout", `{"tier_id":"`+tierID+`"}`, fanID, "fan")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("checkout: want 201 got %d: %s", rec.Code, rec.Body.String())
	}

	var checkoutResp struct {
		TransactionID string  `json:"transaction_id"`
		Amount        float64 `json:"amount"`
		Status        string  `json:"status"`
	}
	json.NewDecoder(rec.Body).Decode(&checkoutResp)
	if checkoutResp.Status != "success" {
		t.Errorf("want success got %s", checkoutResp.Status)
	}

	// Balance should be 9.99.
	bh := applyAuthIntegration(app.GetBalance)
	brec := httptest.NewRecorder()
	bh.ServeHTTP(brec, authReq(http.MethodGet, "/balance", "", creatorID, "creator"))
	if brec.Code != http.StatusOK {
		t.Fatalf("balance: want 200 got %d", brec.Code)
	}
	var balResp struct {
		Available   float64 `json:"available"`
		TotalEarned float64 `json:"total_earned"`
	}
	json.NewDecoder(brec.Body).Decode(&balResp)
	if balResp.Available != 9.99 {
		t.Errorf("balance available: want 9.99 got %v", balResp.Available)
	}
}

func TestIntegration_PayoutFlow(t *testing.T) {
	pool := integrationPool(t)
	app := integrationApp(t, pool)

	// First checkout to add balance.
	ch := applyAuthIntegration(app.Checkout)
	ch.ServeHTTP(httptest.NewRecorder(), authReq(http.MethodPost, "/checkout", `{"tier_id":"`+tierID+`"}`, fanID, "fan"))

	// Create payout.
	ph := applyAuthIntegration(app.CreatePayout)
	prec := httptest.NewRecorder()
	ph.ServeHTTP(prec, authReq(http.MethodPost, "/payouts", `{"amount":9.99}`, creatorID, "creator"))
	if prec.Code != http.StatusCreated {
		t.Fatalf("payout: want 201 got %d: %s", prec.Code, prec.Body.String())
	}

	// Balance available should now be 0.
	bh := applyAuthIntegration(app.GetBalance)
	brec := httptest.NewRecorder()
	bh.ServeHTTP(brec, authReq(http.MethodGet, "/balance", "", creatorID, "creator"))
	var balResp struct {
		Available float64 `json:"available"`
		TotalPaid float64 `json:"total_paid"`
	}
	json.NewDecoder(brec.Body).Decode(&balResp)
	if balResp.Available != 0 {
		t.Errorf("want 0 available got %v", balResp.Available)
	}
	if balResp.TotalPaid != 9.99 {
		t.Errorf("want 9.99 total_paid got %v", balResp.TotalPaid)
	}
}

func TestIntegration_ListTransactions(t *testing.T) {
	pool := integrationPool(t)
	app := integrationApp(t, pool)

	// Checkout to create a transaction.
	ch := applyAuthIntegration(app.Checkout)
	ch.ServeHTTP(httptest.NewRecorder(), authReq(http.MethodPost, "/checkout", `{"tier_id":"`+tierID+`"}`, fanID, "fan"))

	lh := applyAuthIntegration(app.ListTransactions)
	lrec := httptest.NewRecorder()
	lh.ServeHTTP(lrec, authReq(http.MethodGet, "/transactions", "", fanID, "fan"))
	if lrec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", lrec.Code)
	}
	var resp struct {
		Total int64 `json:"total"`
	}
	json.NewDecoder(lrec.Body).Decode(&resp)
	if resp.Total < 1 {
		t.Errorf("want >=1 transaction got %d", resp.Total)
	}
}
