package middlewares

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	"github.com/redis/go-redis/v9"
)

func AuthMiddleware(jwtSecret string, redisClient *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			authHeader := request.Header.Get("Authorization")
			if authHeader == "" || len(authHeader) < 7 {
				http.Error(responseWriter, "Missing token", http.StatusUnauthorized)
				return
			}
			tokenString := authHeader[7:]

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
			result, err := redisClient.Get(request.Context(), tokenString).Result()
			if result != userID || err != nil {
				http.Error(responseWriter, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(request.Context(), "userID", userID)
			next.ServeHTTP(responseWriter, request.WithContext(ctx))
		})
	}
}
