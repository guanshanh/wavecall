package dispatch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/guanshanh/wavecall/internal/auth"
)

func TestDispatchMissingRoom(t *testing.T) {
	tab, _ := LoadTable(map[string]Node{"n1": {Signaling: "a:18080"}})
	h := NewHandler(tab, nil)
	req := httptest.NewRequest(http.MethodGet, "/dispatch", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rr.Code)
	}
}

func TestDispatchOK(t *testing.T) {
	tab, _ := LoadTable(map[string]Node{"n1": {Signaling: "a:18080"}})
	h := NewHandler(tab, nil)
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
	h := NewHandler(&Table{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/dispatch?room=r1", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("code %d", rr.Code)
	}
}

func TestHealth(t *testing.T) {
	tab, _ := LoadTable(map[string]Node{"n1": {Signaling: "a:18080"}})
	h := NewHandler(tab, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code %d", rr.Code)
	}
}

func TestLoginSuccessAndFailures(t *testing.T) {
	users, err := auth.NewDirectory("top-secret", []auth.User{{Account: "achi", Password: "pw"}})
	if err != nil {
		t.Fatal(err)
	}
	tab, err := LoadTable(map[string]Node{"n1": {Signaling: "a:18080"}})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(tab, users)

	okReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"account":"achi","password":"pw"}`))
	okRR := httptest.NewRecorder()
	h.ServeHTTP(okRR, okReq)
	if okRR.Code != http.StatusOK {
		t.Fatalf("success code %d body %s", okRR.Code, okRR.Body.String())
	}
	if okRR.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("cors %q", okRR.Header().Get("Access-Control-Allow-Origin"))
	}
	var okBody struct {
		Token   string `json:"token"`
		Account string `json:"account"`
	}
	if err := json.NewDecoder(okRR.Body).Decode(&okBody); err != nil {
		t.Fatal(err)
	}
	if okBody.Account != "achi" || okBody.Token == "" {
		t.Fatalf("body %+v", okBody)
	}
	if _, err := users.Verify(okBody.Token, time.Now()); err != nil {
		t.Fatalf("issued token: %v", err)
	}

	wrong := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"account":"achi","password":"no"}`))
	wrongRR := httptest.NewRecorder()
	h.ServeHTTP(wrongRR, wrong)
	missing := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"account":"nope","password":"pw"}`))
	missingRR := httptest.NewRecorder()
	h.ServeHTTP(missingRR, missing)
	if wrongRR.Code != http.StatusUnauthorized || missingRR.Code != http.StatusUnauthorized {
		t.Fatalf("codes %d %d", wrongRR.Code, missingRR.Code)
	}
	if wrongRR.Body.String() != missingRR.Body.String() {
		t.Fatalf("401 bodies differ: %q vs %q", wrongRR.Body.String(), missingRR.Body.String())
	}

	empty := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"account":"","password":"pw"}`))
	emptyRR := httptest.NewRecorder()
	h.ServeHTTP(emptyRR, empty)
	if emptyRR.Code != http.StatusBadRequest {
		t.Fatalf("empty account code %d", emptyRR.Code)
	}
	emptyPassword := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"account":"achi","password":""}`))
	emptyPasswordRR := httptest.NewRecorder()
	h.ServeHTTP(emptyPasswordRR, emptyPassword)
	if emptyPasswordRR.Code != http.StatusBadRequest {
		t.Fatalf("empty password code %d", emptyPasswordRR.Code)
	}

	badJSON := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{`))
	badRR := httptest.NewRecorder()
	h.ServeHTTP(badRR, badJSON)
	if badRR.Code != http.StatusBadRequest {
		t.Fatalf("bad json code %d", badRR.Code)
	}

	opt := httptest.NewRequest(http.MethodOptions, "/login", nil)
	optRR := httptest.NewRecorder()
	h.ServeHTTP(optRR, opt)
	if optRR.Code != http.StatusNoContent {
		t.Fatalf("options %d", optRR.Code)
	}
	if optRR.Header().Get("Access-Control-Allow-Methods") != "POST, OPTIONS" {
		t.Fatalf("allow methods %q", optRR.Header().Get("Access-Control-Allow-Methods"))
	}
	if optRR.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Fatalf("allow headers %q", optRR.Header().Get("Access-Control-Allow-Headers"))
	}
}
