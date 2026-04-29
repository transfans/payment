package handlers

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/transfans/payment/internal/profile"
)

// TxBeginner abstracts pgxpool.Pool for transaction management.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// ProfileClient abstracts profile.Client.
type ProfileClient interface {
	GetTier(ctx context.Context, tierID string) (*profile.Tier, error)
	CheckSubscription(ctx context.Context, fanID, creatorID string) (*profile.SubscriptionCheck, error)
}

// MQPublisher abstracts mq.Publisher.
type MQPublisher interface {
	Publish(ctx context.Context, routingKey, requestID string, data any) error
}
