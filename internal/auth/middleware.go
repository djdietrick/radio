package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey int

const claimsKey ctxKey = iota

// FromContext returns the authenticated user's claims, if any.
func FromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

// UserID returns the authenticated user id, or "" if unauthenticated.
func UserID(ctx context.Context) string {
	if c, ok := FromContext(ctx); ok {
		return c.UserID
	}
	return ""
}

// tokenFromRequest extracts a bearer token from the Authorization header, or
// falls back to a `token` query parameter. The query fallback exists for the
// WebSocket route, since browser WebSocket clients cannot set headers.
func tokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if after, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(after)
		}
	}
	return r.URL.Query().Get("token")
}

// Require is middleware that rejects requests without a valid token, attaching
// the verified claims to the request context on success.
func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := tokenFromRequest(r)
		if tok == "" {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		claims, err := a.Verify(tok)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin is middleware that additionally requires the admin flag. It must
// be composed inside Require (which populates the claims).
func (a *Authenticator) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := FromContext(r.Context())
		if !ok || !claims.IsAdmin {
			http.Error(w, "admin privileges required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
