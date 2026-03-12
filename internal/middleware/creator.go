package middleware

import (
	"net/http"

	"github.com/transfans/payment/internal/handlers"
)

func CreatorOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaims(r.Context())
		if !ok || !claims.IsCreator {
			handlers.WriteError(w, http.StatusForbidden, "creator account required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
