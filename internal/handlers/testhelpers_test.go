package handlers_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/transfans/payment/internal/db"
	"github.com/transfans/payment/internal/handlers"
	"github.com/transfans/payment/internal/profile"
)

const testJWTSecret = "test-secret"

func makeJWT(userID, role string) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	s, _ := tok.SignedString([]byte(testJWTSecret))
	return s
}

func newApp(q db.Querier, pc handlers.ProfileClient, pub handlers.MQPublisher, pool handlers.TxBeginner) *handlers.App {
	return &handlers.App{
		Pool:          pool,
		Queries:       q,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Publisher:     pub,
		ProfileClient: pc,
	}
}

func authReq(method, target, body, userID, role string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	r.Header.Set("Authorization", "Bearer "+makeJWT(userID, role))
	return r
}

// --- mock Querier ---

type mockQuerier struct {
	insertTransaction func(context.Context, db.InsertTransactionParams) (db.Transaction, error)
	upsertBalance     func(context.Context, db.UpsertBalanceParams) (db.Balance, error)
	decrementBalance  func(context.Context, db.DecrementBalanceParams) (db.Balance, error)
	getBalance        func(context.Context, pgtype.UUID) (db.Balance, error)
	countTransactions func(context.Context, pgtype.UUID) (int64, error)
	listTransactions  func(context.Context, db.ListTransactionsByFanParams) ([]db.Transaction, error)
	insertPayout      func(context.Context, db.InsertPayoutParams) (db.Payout, error)
	countPayouts      func(context.Context, pgtype.UUID) (int64, error)
	listPayouts       func(context.Context, db.ListPayoutsByCreatorParams) ([]db.Payout, error)
	getRevenue        func(context.Context, db.GetRevenueParams) (pgtype.Numeric, error)
	getRevenueByTier  func(context.Context, db.GetRevenueByTierParams) ([]db.GetRevenueByTierRow, error)
}

func (m *mockQuerier) InsertTransaction(ctx context.Context, arg db.InsertTransactionParams) (db.Transaction, error) {
	return m.insertTransaction(ctx, arg)
}
func (m *mockQuerier) UpsertBalance(ctx context.Context, arg db.UpsertBalanceParams) (db.Balance, error) {
	return m.upsertBalance(ctx, arg)
}
func (m *mockQuerier) DecrementBalance(ctx context.Context, arg db.DecrementBalanceParams) (db.Balance, error) {
	return m.decrementBalance(ctx, arg)
}
func (m *mockQuerier) GetBalance(ctx context.Context, id pgtype.UUID) (db.Balance, error) {
	return m.getBalance(ctx, id)
}
func (m *mockQuerier) CountTransactionsByFan(ctx context.Context, id pgtype.UUID) (int64, error) {
	return m.countTransactions(ctx, id)
}
func (m *mockQuerier) ListTransactionsByFan(ctx context.Context, arg db.ListTransactionsByFanParams) ([]db.Transaction, error) {
	return m.listTransactions(ctx, arg)
}
func (m *mockQuerier) InsertPayout(ctx context.Context, arg db.InsertPayoutParams) (db.Payout, error) {
	return m.insertPayout(ctx, arg)
}
func (m *mockQuerier) CountPayoutsByCreator(ctx context.Context, id pgtype.UUID) (int64, error) {
	return m.countPayouts(ctx, id)
}
func (m *mockQuerier) ListPayoutsByCreator(ctx context.Context, arg db.ListPayoutsByCreatorParams) ([]db.Payout, error) {
	return m.listPayouts(ctx, arg)
}
func (m *mockQuerier) GetRevenue(ctx context.Context, arg db.GetRevenueParams) (pgtype.Numeric, error) {
	return m.getRevenue(ctx, arg)
}
func (m *mockQuerier) GetRevenueByTier(ctx context.Context, arg db.GetRevenueByTierParams) ([]db.GetRevenueByTierRow, error) {
	return m.getRevenueByTier(ctx, arg)
}
func (m *mockQuerier) WithTx(_ pgx.Tx) db.Querier { return m }

// --- mock ProfileClient ---

type mockProfileClient struct {
	getTier           func(context.Context, string) (*profile.Tier, error)
	checkSubscription func(context.Context, string, string) (*profile.SubscriptionCheck, error)
}

func (m *mockProfileClient) GetTier(ctx context.Context, id string) (*profile.Tier, error) {
	return m.getTier(ctx, id)
}
func (m *mockProfileClient) CheckSubscription(ctx context.Context, fanID, creatorID string) (*profile.SubscriptionCheck, error) {
	return m.checkSubscription(ctx, fanID, creatorID)
}

// --- mock Publisher ---

type mockPublisher struct {
	publish func(context.Context, string, string, any) error
}

func (m *mockPublisher) Publish(ctx context.Context, key, reqID string, data any) error {
	return m.publish(ctx, key, reqID, data)
}

// --- fakeTx satisfies pgx.Tx with no-op implementations ---

type fakeTx struct{}

func (fakeTx) Begin(ctx context.Context) (pgx.Tx, error)    { return fakeTx{}, nil }
func (fakeTx) Commit(ctx context.Context) error              { return nil }
func (fakeTx) Rollback(ctx context.Context) error            { return nil }
func (fakeTx) CopyFrom(_ context.Context, _ pgx.Identifier, _ []string, _ pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (fakeTx) SendBatch(_ context.Context, _ *pgx.Batch) pgx.BatchResults { return nil }
func (fakeTx) LargeObjects() pgx.LargeObjects                              { return pgx.LargeObjects{} }
func (fakeTx) Prepare(_ context.Context, _, _ string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (fakeTx) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (fakeTx) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) { return nil, nil }
func (fakeTx) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row        { return nil }
func (fakeTx) Conn() *pgx.Conn                                               { return nil }

// mockPool always returns a fakeTx.
type mockPool struct{}

func (mockPool) Begin(ctx context.Context) (pgx.Tx, error) { return fakeTx{}, nil }
