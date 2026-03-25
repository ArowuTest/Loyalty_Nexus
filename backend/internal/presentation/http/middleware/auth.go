package middleware

import (
	"context"
	"net/http"
	"strings"
	"os"
	"github.com/golang-jwt/jwt/v5"
	"loyalty-nexus/pkg/utils"
)

type contextKey string

const UserMSISDNKey contextKey = "user_msisdn"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public routes exclusion
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/v1/auth") || 
		   strings.HasPrefix(path, "/api/v1/recharge/ingest") ||
		   strings.HasPrefix(path, "/api/v1/ussd") ||
		   strings.HasPrefix(path, "/api/v1/recharge/mno-webhook") {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid auth header", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		secret := []byte(os.Getenv("JWT_SECRET"))

		token, err := jwt.ParseWithClaims(tokenStr, &utils.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*utils.TokenClaims)
		if !ok {
			http.Error(w, "Invalid claims", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserMSISDNKey, claims.MSISDN)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
