package middleware

import (
	"net/http"
	"strings"
)

// Internal verifies the X-Internal-Token header matches the expected
// token. It guards GS-only endpoints such as result reporting.
func Internal(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := r.Header.Get("X-Internal-Token")
			if tok == "" || !strings.EqualFold(tok, expectedToken) {
				writeError(w, http.StatusUnauthorized, "invalid internal token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
