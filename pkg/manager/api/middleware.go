package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// APIKeyAuth returns middleware that validates the Authorization: Bearer <key> header.
func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing authorization header"})
				return
			}

			token, found := strings.CutPrefix(auth, "Bearer ")
			if !found {
				writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid authorization format, expected Bearer <token>"})
				return
			}

			if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
				writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "invalid api key"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
