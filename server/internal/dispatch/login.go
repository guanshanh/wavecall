package dispatch

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.users == nil {
		http.Error(w, "users not configured", http.StatusInternalServerError)
		return
	}
	var body struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Account) == "" || body.Password == "" {
		writeLoginJSON(w, http.StatusBadRequest, "invalid")
		return
	}
	account, ok := h.users.Authenticate(body.Account, body.Password)
	if !ok {
		writeLoginJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	token, err := h.users.Issue(account, time.Now())
	if err != nil {
		http.Error(w, "login failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"token":   token,
		"account": account,
	})
}

func writeLoginJSON(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
