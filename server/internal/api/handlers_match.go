package apipackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.compackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openpackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); errpackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode]package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	tpackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueuepackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, httppackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket":package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket": ticket})
}

func (h *Handlers) cancelMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtxpackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket": ticket})
}

func (h *Handlers) cancelMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	h.Match.Cancel(uid)
	writeJSON(w, http.StatusOK, map[string]bool{"cancelledpackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket": ticket})
}

func (h *Handlers) cancelMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	h.Match.Cancel(uid)
	writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
}

func (h *Handlers) getMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFrompackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket": ticket})
}

func (h *Handlers) cancelMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	h.Match.Cancel(uid)
	writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
}

func (h *Handlers) getMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	matchID := chi.URLParam(r, "id")

	var mode string
	var endedAt, startedAt *time.Time
	varpackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket": ticket})
}

func (h *Handlers) cancelMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	h.Match.Cancel(uid)
	writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
}

func (h *Handlers) getMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	matchID := chi.URLParam(r, "id")

	var mode string
	var endedAt, startedAt *time.Time
	var placement, kills, deaths *int
	err := h.Store.Pool.QueryRow(r.Context(), `
	package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket": ticket})
}

func (h *Handlers) cancelMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	h.Match.Cancel(uid)
	writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
}

func (h *Handlers) getMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	matchID := chi.URLParam(r, "id")

	var mode string
	var endedAt, startedAt *time.Time
	var placement, kills, deaths *int
	err := h.Store.Pool.QueryRow(r.Context(), `
		SELECT m.mode, m.ended_at, m.started_at, mp.placement, mp.kills, mp.deaths
		FROM matches m
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket": ticket})
}

func (h *Handlers) cancelMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	h.Match.Cancel(uid)
	writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
}

func (h *Handlers) getMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	matchID := chi.URLParam(r, "id")

	var mode string
	var endedAt, startedAt *time.Time
	var placement, kills, deaths *int
	err := h.Store.Pool.QueryRow(r.Context(), `
		SELECT m.mode, m.ended_at, m.started_at, mp.placement, mp.kills, mp.deaths
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE m.id = $1 AND mp.user_id = $2`,package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket": ticket})
}

func (h *Handlers) cancelMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	h.Match.Cancel(uid)
	writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
}

func (h *Handlers) getMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	matchID := chi.URLParam(r, "id")

	var mode string
	var endedAt, startedAt *time.Time
	var placement, kills, deaths *int
	err := h.Store.Pool.QueryRow(r.Context(), `
		SELECT m.mode, m.ended_at, m.started_at, mp.placement, mp.kills, mp.deaths
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE m.id = $1 AND mp.user_id = $2`, matchID, uid).
		Scan(&mode, &endedAt, &startedAt, &package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket": ticket})
}

func (h *Handlers) cancelMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	h.Match.Cancel(uid)
	writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
}

func (h *Handlers) getMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	matchID := chi.URLParam(r, "id")

	var mode string
	var endedAt, startedAt *time.Time
	var placement, kills, deaths *int
	err := h.Store.Pool.QueryRow(r.Context(), `
		SELECT m.mode, m.ended_at, m.started_at, mp.placement, mp.kills, mp.deaths
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE m.id = $1 AND mp.user_id = $2`, matchID, uid).
		Scan(&mode, &endedAt, &startedAt, &placement, &kills, &deaths)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONpackage api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/match"
)

var validModes = map[string]bool{
	"deathmatch": true,
	"tdm":        true,
	"domination": true,
	"adventure":  true,
}

type queueReq struct {
	Mode string `json:"mode"`
}

func (h *Handlers) queueMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	username, _ := auth.UsernameFromCtx(r.Context())

	var req queueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validModes[req.Mode] {
		writeJSONError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	ticket, err := h.Match.Enqueue(uid, username, req.Mode)
	if err != nil {
		if errors.Is(err, match.ErrAlreadyInQueue) {
			writeJSONError(w, http.StatusConflict, "already in queue")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": true, "ticket": ticket})
}

func (h *Handlers) cancelMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	h.Match.Cancel(uid)
	writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
}

func (h *Handlers) getMatch(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())
	matchID := chi.URLParam(r, "id")

	var mode string
	var endedAt, startedAt *time.Time
	var placement, kills, deaths *int
	err := h.Store.Pool.QueryRow(r.Context(), `
		SELECT m.mode, m.ended_at, m.started_at, mp.placement, mp.kills, mp.deaths
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE m.id = $1 AND mp.user_id = $2`, matchID, uid).
		Scan(&mode, &endedAt, &startedAt, &placement, &kills, &deaths)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "match not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error