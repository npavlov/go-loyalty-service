package middlewares

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v4"

	"github.com/npavlov/go-loyalty-service/internal/redis"
)

const UserIDKey string = "userID"

func AuthMiddleware(jwtSecret string, memStorage redis.MemStorage) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			// Retrieve the "Authorization" token
			tokenString := request.Header.Get("Authorization")

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(responseWriter, "Invalid token", http.StatusUnauthorized)

				return
			}

			userID, ok := claims["user_id"].(string)
			if !ok {
				http.Error(responseWriter, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			// Check if the token exists in Redis and match with User ID
			result, err := memStorage.Get(request.Context(), tokenString)
			if result != userID || err != nil {
				http.Error(responseWriter, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(request.Context(), UserIDKey, userID)
			next.ServeHTTP(responseWriter, request.WithContext(ctx))
		})
	}
}
