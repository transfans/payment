package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/transfans/payment/internal/db"
	"github.com/transfans/payment/internal/httputil"
	"github.com/transfans/payment/internal/middleware"
	"github.com/transfans/payment/internal/mq"
	"github.com/transfans/payment/internal/profile"
)

const subscriptionDuration = 30 * 24 * time.Hour

type checkoutRequest struct {
	TierID string `json:"tier_id"`
}

type checkoutResponse struct {
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	ExpiresAt     string  `json:"expires_at"`
}

func (a *App) Checkout(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r.Context())
	requestID := middleware.GetRequestID(r.Context())

	var req checkoutRequest
	if err := httputil.ReadJSON(w, r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.TierID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "tier_id is required")
		return
	}

	tier, err := a.ProfileClient.GetTier(r.Context(), req.TierID)
	if err != nil {
		if errors.Is(err, profile.ErrNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "tier not found")
			return
		}
		a.Logger.Error("get tier", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if !tier.IsActive {
		httputil.WriteError(w, http.StatusNotFound, "tier not found")
		return
	}

	check, err := a.ProfileClient.CheckSubscription(r.Context(), claims.UserID, tier.CreatorID)
	if err != nil {
		a.Logger.Error("check subscription", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if check.HasAccess {
		httputil.WriteError(w, http.StatusConflict, "already subscribed to this creator")
		return
	}

	var fanID pgtype.UUID
	if err := fanID.Scan(claims.UserID); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var creatorID pgtype.UUID
	if err := creatorID.Scan(tier.CreatorID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "invalid creator id")
		return
	}

	var tierID pgtype.UUID
	if err := tierID.Scan(req.TierID); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid tier_id")
		return
	}

	var amount pgtype.Numeric
	if err := amount.Scan(strconv.FormatFloat(tier.Price, 'f', 2, 64)); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "invalid tier price")
		return
	}

	tx, err := a.Queries.InsertTransaction(r.Context(), db.InsertTransactionParams{
		FanID:     fanID,
		CreatorID: creatorID,
		TierID:    tierID,
		Amount:    amount,
		Status:    "success",
	})
	if err != nil {
		a.Logger.Error("insert transaction", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	expiresAt := time.Now().UTC().Add(subscriptionDuration)

	if err := a.Publisher.Publish(r.Context(), mq.RoutingKeySubscriptionCreate, requestID, mq.SubscriptionCreateData{
		FanID:     claims.UserID,
		CreatorID: tier.CreatorID,
		TierID:    req.TierID,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}); err != nil {
		a.Logger.Error("publish subscription.create.request", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, checkoutResponse{
		TransactionID: uuidToString(tx.ID),
		Amount:        numericToFloat64(tx.Amount),
		Status:        tx.Status,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
	})
}
