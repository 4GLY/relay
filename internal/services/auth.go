package services

import "context"

const (
	APIKeyScopeGlobal  = "global"
	APIKeyScopeProject = "project"
)

type AuthInfo struct {
	KeyID     string
	IsAdmin   bool
	Scope     string
	ProjectID string
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
