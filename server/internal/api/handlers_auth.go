package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"openfire-server/internal/auth"
)

type registerReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResp struct {
	Token     string `json:"token"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expires_at"`
}

func (h *Handlers) register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if err := auth.ValidateCredentials(req.Username, req.Password); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	token, uid, exp, err := h.Auth.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrUsernameTaken) {
			writeJSONError(w, http.StatusConflict, "username taken")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, authResp{Token: token, UserID: uid, Username: req.Username, ExpiresAt: exp})
}

func (h *Handlers) login(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, uid, exp, err := h.Auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, authResp{Token: token, UserID: uid, Username: req.Username, ExpiresAt: exp})
}
