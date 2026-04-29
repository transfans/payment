package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/transfans/payment/internal/middleware"
)

const testSecret = "test-secret"

func makeToken(t *testing.T, claims jwt.MapClaims, secret string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		http.Error(w, "no claims", http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-User-ID", claims.UserID)
	w.Header().Set("X-Role", claims.Role)
	w.WriteHeader(http.StatusOK)
}

func TestAuth_MissingHeader(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401 got %d", rec.Code)
	}
}

func TestAuth_WrongScheme(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic sometoken")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401 got %d", rec.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer notavalidtoken")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401 got %d", rec.Code)
	}
}

func TestAuth_WrongSecret(t *testing.T) {
	tok := makeToken(t, jwt.MapClaims{"sub": "u1", "role": "fan"}, "other-secret")
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401 got %d", rec.Code)
	}
}

func TestAuth_ExpiredToken(t *testing.T) {
	tok := makeToken(t, jwt.MapClaims{
		"sub":  "u1",
		"role": "fan",
		"exp":  time.Now().Add(-time.Hour).Unix(),
	}, testSecret)
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401 got %d", rec.Code)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	tok := makeToken(t, jwt.MapClaims{
		"sub":   "user-123",
		"role":  "fan",
		"email": "fan@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}, testSecret)
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200 got %d", rec.Code)
	}
	if got := rec.Header().Get("X-User-ID"); got != "user-123" {
		t.Errorf("want user-123 got %q", got)
	}
	if got := rec.Header().Get("X-Role"); got != "fan" {
		t.Errorf("want fan got %q", got)
	}
}
