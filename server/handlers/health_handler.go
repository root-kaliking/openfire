// Package handlers contains the HTTP handlers for the central server's
// public API, the WS lobby upgrade, and the internal GS result endpoint.
package handlers

import (
	"encoding/json"
	"net/http"
)

// Health answers GET /healthz.
func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// writeJSON encodes v with the given status and a JSON content type.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeJSONError writes {"error":"..."} with the given status. All
// non-2xx responses in the API use this shape.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
