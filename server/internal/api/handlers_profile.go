package api

import (
	"net/http"

	"openfire-server/internal/auth"
)

type statsResp struct {
	Wins    int `json:"wins"`
	Losses  int `json:"losses"`
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Matches int `json:"matches"`
}

func (h *Handlers) profile(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromCtx(r.Context())

	var username string
	if err := h.Store.Pool.QueryRow(r.Context(),
		"SELECT username FROM users WHERE id=$1", uid).Scan(&username); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var st statsResp
	if err := h.Store.Pool.QueryRow(r.Context(),
		"SELECT wins, losses, kills, deaths, matches FROM user_stats WHERE user_id=$1", uid).
		Scan(&st.Wins, &st.Losses, &st.Kills, &st.Deaths, &st.Matches); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":  uid,
		"username": username,
		"stats":    st,
	})
}
