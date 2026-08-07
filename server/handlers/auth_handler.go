package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"openfire-server/internal/auth"
)

// AuthHandler handles registration and login.
type AuthHandler struct {
	db        *sql.DB
	jwtSecret string
	jwtTTL    time.Duration
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(db *sql.DB, jwtSecret string, jwtTTL time.Duration) *AuthHandler {
	return &AuthHandler{db: db, jwtSecret: jwtSecret, jwtTTL: jwtTTL}
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type authResponse struct {
	Token string   `json:"token"`
	User  userInfo `json:"user"`
}

// Register handles POST /api/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" || len(username) > 32 {
		writeJSONError(w, http.StatusBadRequest, "username must be 1-32 chars")
		return
	}
	if len(req.Password) < 6 || len(req.Password) > 128 {
		writeJSONError(w, http.StatusBadRequest, "password must be 6-128 chars")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	id := uuid.NewString()
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO users (id, username, password_hash) VALUES ($1,$2,$3)`,
		id, username, hash); err != nil {
		if isUniqueViolation(err) {
			writeJSONError(w, http.StatusConflict, "username already taken")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	token, err := auth.Sign(h.jwtSecret, id, username, h.jwtTTL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: userInfo{ID: id, Username: username}})
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	username := strings.TrimSpace(req.Username)

	var (
		id   string
		hash string
	)
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, password_hash FROM users WHERE username=$1`, username).Scan(&id, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	if err := auth.CheckPassword(hash, req.Password); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.Sign(h.jwtSecret, id, username, h.jwtTTL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: userInfo{ID: id, Username: username}})
}

// isUniqueViolation detects a Postgres unique-violation (SQLSTATE 23505)
// from the lib/pq driver.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}
