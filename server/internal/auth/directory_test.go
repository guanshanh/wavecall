package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRejectsMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewDirectoryValidation(t *testing.T) {
	cases := []struct {
		name    string
		secret  string
		users   []User
		wantSub string
	}{
		{"empty secret", "  ", []User{{Account: "a", Password: "p"}}, "empty secret"},
		{"no users", "secret", nil, "no users"},
		{"empty password", "secret", []User{{Account: "a", Password: ""}}, "empty password"},
		{"blank account", "secret", []User{{Account: "  ", Password: "p"}}, "invalid account"},
		{"space in account", "secret", []User{{Account: "a b", Password: "p"}}, "invalid account"},
		{"pipe in account", "secret", []User{{Account: "a|b", Password: "p"}}, "invalid account"},
		{"duplicate", "secret", []User{{Account: "a", Password: "p"}, {Account: " a ", Password: "q"}}, "duplicate account"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewDirectory(tc.secret, tc.users)
			if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("err %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}

func TestLoadAndAuthenticate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.toml")
	body := "secret = \" top-secret \"\n\n[[users]]\naccount = \" achi \"\npassword = \" p@ss \"\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	users, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := users.Authenticate("achi", " p@ss ")
	if !ok || got != "achi" {
		t.Fatalf("authenticate match: %q %v", got, ok)
	}
	if _, ok := users.Authenticate("achi", "wrong"); ok {
		t.Fatal("wrong password should fail")
	}
	if _, ok := users.Authenticate("nope", " p@ss "); ok {
		t.Fatal("unknown account should fail")
	}
	if _, ok := users.Authenticate("ACHI", " p@ss "); ok {
		t.Fatal("account match is case-sensitive")
	}
}
