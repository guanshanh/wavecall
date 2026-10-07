package dispatch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDispatchMissingRoom(t *testing.T) {
	tab, _ := LoadTable(map[string]Node{"n1": {Signaling: "a:18080"}})
	h := NewHandler(tab)
	req := httptest.NewRequest(http.MethodGet, "/dispatch", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rr.Code)
	}
}

func TestDispatchOK(t *testing.T) {
	tab, _ := LoadTable(map[string]Node{"n1": {Signaling: "a:18080"}})
	h := NewHandler(tab)
	req := httptest.NewRequest(http.MethodGet, "/dispatch?room=r1", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code %d body %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS header, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	var body struct {
		Node string `json:"node"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Node != "a:18080" {
		t.Fatalf("node %q", body.Node)
	}
}

func TestDispatchEmptyTable(t *testing.T) {
	h := NewHandler(&Table{})
	req := httptest.NewRequest(http.MethodGet, "/dispatch?room=r1", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("code %d", rr.Code)
	}
}

func TestHealth(t *testing.T) {
	tab, _ := LoadTable(map[string]Node{"n1": {Signaling: "a:18080"}})
	h := NewHandler(tab)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code %d", rr.Code)
	}
}
