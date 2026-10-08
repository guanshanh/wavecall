package dispatch

import (
	"encoding/json"
	"net/http"

	"github.com/guanshanh/wavecall/internal/auth"
)

type Handler struct {
	tab   *Table
	users *auth.Directory
}

func NewHandler(tab *Table, users *auth.Directory) http.Handler {
	h := &Handler{tab: tab, users: users}
	mux := http.NewServeMux()
	mux.HandleFunc("/dispatch", h.handleDispatch)
	mux.HandleFunc("/login", h.handleLogin)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func (h *Handler) handleDispatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	room := r.URL.Query().Get("room")
	if room == "" {
		http.Error(w, "missing room", http.StatusBadRequest)
		return
	}
	node, err := h.tab.Lookup(room)
	if err == ErrEmptyTable {
		http.Error(w, "no nodes", http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"node": node})
}
