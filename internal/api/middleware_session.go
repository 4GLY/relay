package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"relay/internal/contracts"
	"relay/internal/lib"
	"relay/internal/services"
)

// requireSessionOrAdmin gates a handler behind two accepted credentials:
//
//  1. Bearer admin token (preserves R1 + admin tooling parity with
//     requireAdminBearerToken).
//  2. Otherwise a relay_session cookie that resolves to a real user.
//
// The handler downstream sees a populated services.AuthInfo on the request
// context. Project ownership is enforced inside the service layer via
// requireAdminOrProjectOwner / enforceProjectAccess so this middleware does
// not need to know about specific resource ids.
func requireSessionOrAdmin(adminToken string, svc services.Service, next http.HandlerFunc) http.HandlerFunc {
	if adminToken == "" {
		return func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, contracts.Failure("api auth", "MISCONFIGURED", "admin token is not configured", false, "RELAY_ADMIN_TOKEN or RELAY_API_TOKEN"))
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
			provided := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			if provided != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(adminToken)) == 1 {
				ctx := services.ContextWithAuthInfo(r.Context(), services.AuthInfo{
					IsAdmin: true,
					Scope:   services.APIKeyScopeGlobal,
				})
				next(w, r.WithContext(ctx))
				return
			}
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeJSON(w, http.StatusUnauthorized, contracts.Failure("api auth", "UNAUTHORIZED", "missing session cookie or admin bearer token", false, "Authorization", sessionCookieName))
			return
		}
		user, err := svc.GetUserBySessionToken(r.Context(), cookie.Value)
		if err != nil {
			if appErr, ok := err.(lib.AppError); ok && appErr.Code == "MISCONFIGURED" {
				writeJSON(w, http.StatusInternalServerError, contracts.Failure("api auth", appErr.Code, appErr.Message, false))
				return
			}
			writeJSON(w, http.StatusUnauthorized, contracts.Failure("api auth", "UNAUTHORIZED", "invalid or expired session", false))
			return
		}
		ctx := services.ContextWithAuthInfo(r.Context(), services.AuthInfo{
			UserID: user.ID,
			Scope:  services.APIKeyScopeGlobal,
		})
		next(w, r.WithContext(ctx))
	}
}
