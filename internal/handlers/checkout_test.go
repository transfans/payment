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
	"github.com/transfans/payment/internal/handlers"
	"github.com/transfans/payment/internal/middleware"
	"github.com/transfans/payment/internal/profile"
)

const (
	fanID     = "00000000-0000-0000-0000-000000000001"
	creatorID = "00000000-0000-0000-0000-000000000002"
	tierID    = "00000000-0000-0000-0000-000000000003"
	txID      = "00000000-0000-0000-0000-000000000004"
)

func successQuerier() *mockQuerier {
	var id pgtype.UUID
	_ = id.Scan(txID)
	var amount pgtype.Numeric
	_ = amount.Scan("9.99")
	return &mockQuerier{
		insertTransaction: func(_ context.Context, _ db.InsertTransactionParams) (db.Transaction, error) {
			return db.Transaction{ID: id, Amount: amount, Status: "success"}, nil
		},
		upsertBalance: func(_ context.Context, _ db.UpsertBalanceParams) (db.Balance, error) {
			return db.Balance{}, nil
		},
	}
}

func successProfileClient() *mockProfileClient {
	return &mockProfileClient{
		getTier: func(_ context.Context, id string) (*profile.Tier, error) {
			return &profile.Tier{ID: id, CreatorID: creatorID, Price: 9.99, IsActive: true}, nil
		},
		checkSubscription: func(_ context.Context, _, _ string) (*profile.SubscriptionCheck, error) {
			return &profile.SubscriptionCheck{HasAccess: false}, nil
		},
	}
}

func successPublisher() *mockPublisher {
	return &mockPublisher{
		publish: func(_ context.Context, _, _ string, _ any) error { return nil },
	}
}

func applyAuth(_ *handlers.App, next http.HandlerFunc) http.Handler {
	return middleware.Auth(testJWTSecret)(next)
}

func TestCheckout_MissingTierID(t *testing.T) {
	app := newApp(successQuerier(), successProfileClient(), successPublisher(), mockPool{})
	h := applyAuth(app, app.Checkout)
	req := authReq(http.MethodPost, "/checkout", `{}`, fanID, "fan")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCheckout_TierNotFound(t *testing.T) {
	pc := &mockProfileClient{
		getTier: func(_ context.Context, _ string) (*profile.Tier, error) {
			return nil, profile.ErrNotFound
		},
	}
	app := newApp(successQuerier(), pc, successPublisher(), mockPool{})
	h := applyAuth(app, app.Checkout)
	req := authReq(http.MethodPost, "/checkout", `{"tier_id":"`+tierID+`"}`, fanID, "fan")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404 got %d", rec.Code)
	}
}

func TestCheckout_TierInactive(t *testing.T) {
	pc := &mockProfileClient{
		getTier: func(_ context.Context, _ string) (*profile.Tier, error) {
			return &profile.Tier{ID: tierID, CreatorID: creatorID, Price: 9.99, IsActive: false}, nil
		},
	}
	app := newApp(successQuerier(), pc, successPublisher(), mockPool{})
	h := applyAuth(app, app.Checkout)
	req := authReq(http.MethodPost, "/checkout", `{"tier_id":"`+tierID+`"}`, fanID, "fan")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404 got %d", rec.Code)
	}
}

func TestCheckout_AlreadySubscribed(t *testing.T) {
	pc := &mockProfileClient{
		getTier: func(_ context.Context, _ string) (*profile.Tier, error) {
			return &profile.Tier{ID: tierID, CreatorID: creatorID, Price: 9.99, IsActive: true}, nil
		},
		checkSubscription: func(_ context.Context, _, _ string) (*profile.SubscriptionCheck, error) {
			return &profile.SubscriptionCheck{HasAccess: true}, nil
		},
	}
	app := newApp(successQuerier(), pc, successPublisher(), mockPool{})
	h := applyAuth(app, app.Checkout)
	req := authReq(http.MethodPost, "/checkout", `{"tier_id":"`+tierID+`"}`, fanID, "fan")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("want 409 got %d", rec.Code)
	}
}

func TestCheckout_Success(t *testing.T) {
	app := newApp(successQuerier(), successProfileClient(), successPublisher(), mockPool{})
	h := applyAuth(app, app.Checkout)
	req := authReq(http.MethodPost, "/checkout", `{"tier_id":"`+tierID+`"}`, fanID, "fan")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("want 201 got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		TransactionID string  `json:"transaction_id"`
		Amount        float64 `json:"amount"`
		Status        string  `json:"status"`
		ExpiresAt     string  `json:"expires_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.TransactionID != txID {
		t.Errorf("want txID %s got %s", txID, resp.TransactionID)
	}
	if resp.Status != "success" {
		t.Errorf("want success got %s", resp.Status)
	}
	// ExpiresAt should be ~30 days from now
	exp, err := time.Parse(time.RFC3339, resp.ExpiresAt)
	if err != nil {
		t.Fatalf("parse expires_at: %v", err)
	}
	if d := time.Until(exp); d < 29*24*time.Hour || d > 31*24*time.Hour {
		t.Errorf("unexpected expires_at: %v", exp)
	}
}

func TestCheckout_NoAuth(t *testing.T) {
	app := newApp(successQuerier(), successProfileClient(), successPublisher(), mockPool{})
	h := applyAuth(app, app.Checkout)
	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401 got %d", rec.Code)
	}
}
