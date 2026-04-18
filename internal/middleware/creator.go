package middleware

import (
	"net/http"

	"github.com/transfans/payment/internal/httputil"
)

func CreatorOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaims(r.Context())
		if !ok || claims.Role != "creator" {
			httputil.WriteError(w, http.StatusForbidden, "creator account required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
