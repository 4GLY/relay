package services

import (
	"context"

	"relay/internal/lib"
)

const (
	APIKeyScopeGlobal  = "global"
	APIKeyScopeProject = "project"
)

type AuthInfo struct {
	KeyID     string
	UserID    string
	IsAdmin   bool
	Scope     string
	ProjectID string
}

// RequireUserAuth returns AuthInfo when the request carries a populated user
// identity (typically via the cookie session middleware). Otherwise it returns
// a FORBIDDEN error so handlers can refuse with 401/403 semantics.
func RequireUserAuth(ctx context.Context) (AuthInfo, error) {
	auth, ok := AuthInfoFromContext(ctx)
	if !ok || auth.UserID == "" {
		return AuthInfo{}, lib.Forbidden("UNAUTHORIZED", "user authentication required")
	}
	return auth, nil
}

type authContextKey struct{}

func ContextWithAuthInfo(ctx context.Context, info AuthInfo) context.Context {
	info.Scope = NormalizeAPIKeyScope(info.Scope)
	return context.WithValue(ctx, authContextKey{}, info)
}

func AuthInfoFromContext(ctx context.Context) (AuthInfo, bool) {
	info, ok := ctx.Value(authContextKey{}).(AuthInfo)
	if !ok {
		return AuthInfo{}, false
	}
	info.Scope = NormalizeAPIKeyScope(info.Scope)
	return info, true
}

func NormalizeAPIKeyScope(scope string) string {
	switch scope {
	case "", APIKeyScopeGlobal:
		return APIKeyScopeGlobal
	case APIKeyScopeProject:
		return APIKeyScopeProject
	default:
		return scope
	}
}

func IsKnownAPIKeyScope(scope string) bool {
	switch NormalizeAPIKeyScope(scope) {
	case APIKeyScopeGlobal, APIKeyScopeProject:
		return true
	default:
		return false
	}
}
