package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// Principal is the authenticated subject placed on the request context.
type Principal struct {
	UserID int64
	Role   string
	Alias  string
}

type ctxKey int

const principalKey ctxKey = 1

// TokenParser validates a Bearer access token.
type TokenParser interface {
	ParseAccessToken(accessToken string) (userID int64, role, alias string, err error)
}

// PrincipalFromContext returns the auth principal if present.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

// ContextWithPrincipal stores a principal (tests / handlers).
func ContextWithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// RequireAuth rejects requests without a valid Bearer token.
func RequireAuth(parser TokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			userID, role, alias, err := parser.ParseAccessToken(token)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			ctx := ContextWithPrincipal(r.Context(), Principal{
				UserID: userID,
				Role:   role,
				Alias:  alias,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole rejects authenticated principals that lack role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFromContext(r.Context())
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if p.Role != role {
				writeJSONError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, prefix))
}
