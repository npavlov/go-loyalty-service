package middlewares_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/npavlov/go-loyalty-service/internal/middlewares"
	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
	"github.com/stretchr/testify/assert"
)

const jwtSecret = "test-secret"

func generateJWT(userID string, secret string, expiration time.Duration) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(expiration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString([]byte(secret))
	return signedToken
}

func TestAuthMiddleware(t *testing.T) {
	mockRedis := testutils.NewMockRedis()
	userID := "user123"
	validToken := generateJWT(userID, jwtSecret, time.Minute)
	expiredToken := generateJWT(userID, jwtSecret, -time.Minute)

	// Set valid token in Redis
	_ = mockRedis.Set(context.Background(), validToken, userID, time.Minute)

	// Handler to test
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value("userID")
		assert.Equal(t, userID, user, "Expected userID in context")
		w.WriteHeader(http.StatusOK)
	})

	// Middleware with mock Redis
	middleware := middlewares.AuthMiddleware(jwtSecret, mockRedis)

	t.Run("Valid Token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", validToken)
		rec := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "Valid token should pass")
	})

	t.Run("Invalid Token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "invalid-token")
		rec := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code, "Invalid token should be rejected")
		assert.Contains(t, rec.Body.String(), "Invalid token")
	})

	t.Run("Expired Token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", expiredToken)
		rec := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code, "Expired token should be rejected")
		assert.Contains(t, rec.Body.String(), "Invalid token")
	})

	t.Run("Token Not in Redis", func(t *testing.T) {
		tokenNotInRedis := generateJWT("user456", jwtSecret, time.Minute)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", tokenNotInRedis)
		rec := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code, "Token not in Redis should be rejected")
		assert.Contains(t, rec.Body.String(), "Invalid or expired token")
	})

	t.Run("Redis Token Mismatch", func(t *testing.T) {
		mismatchedToken := generateJWT("user456", jwtSecret, time.Minute)
		_ = mockRedis.Set(context.Background(), mismatchedToken, "differentUser", time.Minute)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", mismatchedToken)
		rec := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code, "Mismatched Redis token should be rejected")
		assert.Contains(t, rec.Body.String(), "Invalid or expired token")
	})
}
