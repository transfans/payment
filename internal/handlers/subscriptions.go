package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/transfans/payment/internal/httputil"
	"github.com/transfans/payment/internal/middleware"
	"github.com/transfans/payment/internal/mq"
)

type cancelSubscriptionRequest struct {
	Reason *string `json:"reason"`
}

func (a *App) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID := chi.URLParam(r, "id")
	if subscriptionID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "subscription id is required")
		return
	}

	requestID := middleware.GetRequestID(r.Context())

	var req cancelSubscriptionRequest
	_ = httputil.ReadJSON(w, r, &req)

	if err := a.Publisher.Publish(r.Context(), mq.RoutingKeySubscriptionDeactivate, requestID, mq.SubscriptionDeactivateData{
		SubscriptionID: subscriptionID,
		Reason:         req.Reason,
	}); err != nil {
		a.Logger.Error("publish subscription.deactivate.request", "error", err, "subscription_id", subscriptionID)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
