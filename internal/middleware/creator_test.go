package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/transfans/payment/internal/middleware"
)

func TestCreatorOnly_NoToken(t *testing.T) {
	h := middleware.CreatorOnly(http.HandlerFunc(okHandler))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403 got %d", rec.Code)
	}
}

func TestCreatorOnly_FanRole(t *testing.T) {
	tok := makeToken(t, jwt.MapClaims{"sub": "u1", "role": "fan", "exp": time.Now().Add(time.Hour).Unix()}, testSecret)
	// Wrap with Auth so claims are in context, then CreatorOnly
	h := middleware.Auth(testSecret)(middleware.CreatorOnly(http.HandlerFunc(okHandler)))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403 got %d", rec.Code)
	}
}

func TestCreatorOnly_CreatorRole(t *testing.T) {
	tok := makeToken(t, jwt.MapClaims{"sub": "u1", "role": "creator", "exp": time.Now().Add(time.Hour).Unix()}, testSecret)
	h := middleware.Auth(testSecret)(middleware.CreatorOnly(http.HandlerFunc(okHandler)))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200 got %d", rec.Code)
	}
}
