// Package identity carries the authenticated user id and API-key scopes
// across request context without creating import cycles between auth and
// feature packages.
package identity

import "context"

type ctxKey string

const (
	Key        ctxKey = "user_id"
	ScopesKey  ctxKey = "api_key_scopes"
	AuthViaKey ctxKey = "auth_via_api_key"
)

func WithUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, Key, id)
}

// UserID returns the authenticated user id, or nil for anonymous requests.
func UserID(ctx context.Context) *int64 {
	if v, ok := ctx.Value(Key).(int64); ok {
		return &v
	}
	return nil
}

// WithScopes marks the request as authenticated via API key with the given
// scope set.
func WithScopes(ctx context.Context, scopes []string) context.Context {
	set := make(map[string]struct{}, len(scopes))
	for _, s := range scopes {
		set[s] = struct{}{}
	}
	ctx = context.WithValue(ctx, ScopesKey, set)
	return context.WithValue(ctx, AuthViaKey, true)
}

// ViaAPIKey reports whether authentication used an API key (and therefore
// scopes apply).
func ViaAPIKey(ctx context.Context) bool {
	v, _ := ctx.Value(AuthViaKey).(bool)
	return v
}

// HasScope reports whether the request may perform the named scope.
// Session-authenticated and anonymous requests carry no scope set and are
// allowed (route-level auth/anonymous policy decides the rest); API-key
// requests must hold the exact scope.
func HasScope(ctx context.Context, scope string) bool {
	if !ViaAPIKey(ctx) {
		return true
	}
	set, ok := ctx.Value(ScopesKey).(map[string]struct{})
	if !ok {
		return false
	}
	_, has := set[scope]
	return has
}
