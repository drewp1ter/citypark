package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/drewp1ter/citypark-go/internal/httpx"
)

// Identity is the signed-in user of a request.
type Identity struct {
	UserID int64
	Role   string
}

type ctxKey struct{}

// NewContext returns ctx carrying id. Used by the middleware and by tests.
func NewContext(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext returns the identity stored by RequireRole.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok
}

// Allows reports whether a user with role `have` may use a route that needs role `need`.
func Allows(have, need string) bool {
	switch need {
	case RoleUser:
		return have == RoleUser || have == RoleAdmin
	case RoleAdmin:
		return have == RoleAdmin
	}
	return false
}

// RequireRole returns a middleware for routes that need the given role.
func (t *Tokens) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				httpx.WriteError(w, r, httpx.Unauthorized())
				return
			}
			claims, err := t.Parse(token)
			if err != nil {
				httpx.WriteError(w, r, httpx.Unauthorized())
				return
			}
			if !Allows(claims.Role, role) {
				httpx.WriteError(w, r, httpx.Forbidden())
				return
			}
			ctx := NewContext(r.Context(), Identity{UserID: claims.UserID, Role: claims.Role})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
