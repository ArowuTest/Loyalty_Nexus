// Package middleware provides HTTP middleware for the Loyalty Nexus API server.
package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
)

type contextKey string
const (
	ContextUserID      contextKey = "user_id"
	ContextPhone       contextKey = "phone"
	ContextIsAdmin     contextKey = "is_admin"
	ContextAdminRole   contextKey = "admin_role"
	ContextAdminClaims contextKey = "admin_claims"
)

// AuthMiddleware validates the Bearer JWT token (user) and injects claims into context.
func AuthMiddleware(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			claims, err := authSvc.ValidateJWT(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextPhone, claims.PhoneNumber)
			ctx = context.WithValue(ctx, ContextIsAdmin, false)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware is like AuthMiddleware but non-blocking: if no token is
// present (or the token is invalid), the request proceeds unauthenticated.
// Handlers read the injected user_id with middleware.ContextUserID; it will be ""
// for unauthenticated requests. Used on public endpoints that benefit from knowing
// the caller's identity when they happen to be logged in (e.g. /recharge/initiate).
func OptionalAuthMiddleware(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token != "" {
				if claims, err := authSvc.ValidateJWT(token); err == nil {
					ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
					ctx = context.WithValue(ctx, ContextPhone, claims.PhoneNumber)
					ctx = context.WithValue(ctx, ContextIsAdmin, false)
					r = r.WithContext(ctx)
				}
				// Invalid token on a public route — silently ignore, proceed as guest
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AdminAuthMiddleware validates admin JWT tokens (email+password issued).
// It injects the full JWTClaims so handlers can check RBAC roles.
func AdminAuthMiddleware(adminAuthSvc *services.AdminAuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			claims, err := adminAuthSvc.ValidateAdminJWT(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired admin token")
				return
			}
			if !claims.IsAdmin {
				writeError(w, http.StatusForbidden, "admin access required")
				return
			}
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextIsAdmin, true)
			ctx = context.WithValue(ctx, ContextAdminRole, string(claims.Role))
			ctx = context.WithValue(ctx, ContextAdminClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns 403 if the caller's role is not in the allowed list.
// Use inside handlers: middleware.RequireRole(w, r, entities.RoleSuperAdmin, entities.RoleFinance)
func RequireRole(w http.ResponseWriter, r *http.Request, roles ...entities.AdminRole) bool {
	role, _ := r.Context().Value(ContextAdminRole).(string)
	for _, allowed := range roles {
		if string(allowed) == role {
			return true
		}
	}
	writeError(w, http.StatusForbidden, "insufficient role for this action")
	return false
}

// allowedOrigins lists every origin permitted to call the API.
// Add new frontend domains here — never use * in production.
var allowedOrigins = map[string]bool{
	"https://loyalty-nexus.vercel.app":       true,
	"https://loyalty-nexus-admin.vercel.app": true,
	"http://localhost:3000":                  true,
	"http://localhost:3001":                  true,
	"http://localhost:8080":                  true,
}

// CORS middleware — restricts requests to known frontend origins.
// Wildcard (*) is never used; each preflight echoes the requesting origin back
// only if it is in the allowedOrigins map.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequestLogger logs each HTTP request.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	// EventSource (SSE) cannot set the Authorization header, so accept ?token=
	// as a fallback — but ONLY for GET requests that explicitly declare they want
	// a server-sent event stream (Accept: text/event-stream).  This prevents
	// tokens from leaking into server access logs on normal API calls where the
	// caller simply forgot to set the Authorization header.
	if r.Method == http.MethodGet &&
		strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		if t := r.URL.Query().Get("token"); t != "" {
			return t
		}
	}
	return ""
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if encErr := json.NewEncoder(w).Encode(map[string]string{"error": msg}); encErr != nil {
		log.Printf("[Auth] writeError encode failure: %v", encErr)
	}
}
