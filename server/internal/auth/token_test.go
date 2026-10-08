package auth

import (
	"strings"
	"testing"
	"time"
)

func testDir(t *testing.T) *Directory {
	t.Helper()
	dir, err := NewDirectory("top-secret", []User{{Account: "achi", Password: "pw"}})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestIssueAndVerifyRoundTrip(t *testing.T) {
	dir := testDir(t)
	now := time.Unix(1_700_000_000, 0)
	token, err := dir.Issue("achi", now)
	if err != nil {
		t.Fatal(err)
	}
	account, err := dir.Verify(token, now)
	if err != nil || account != "achi" {
		t.Fatalf("verify %q err %v", account, err)
	}
}

func TestVerifyExpiryBoundary(t *testing.T) {
	dir := testDir(t)
	now := time.Unix(1_700_000_000, 0)
	token, err := dir.Issue("achi", now)
	if err != nil {
		t.Fatal(err)
	}
	exp := now.Add(TokenTTL).Unix()
	if _, err := dir.Verify(token, time.Unix(exp, 0)); err != nil {
		t.Fatalf("exp instant should be valid: %v", err)
	}
	if _, err := dir.Verify(token, time.Unix(exp+1, 0)); err != ErrInvalidToken {
		t.Fatalf("second after exp: %v", err)
	}
}

func TestVerifyRejectsTamperAndShape(t *testing.T) {
	dir := testDir(t)
	token, err := dir.Issue("achi", time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	bad := []string{"", "abc", "a.b.c", parts[0] + ".aaaa", "aaaa." + parts[1]}
	for _, token := range bad {
		if _, err := dir.Verify(token, time.Unix(1_700_000_000, 0)); err != ErrInvalidToken {
			t.Fatalf("token %q err %v", token, err)
		}
	}
}

func TestVerifyOtherSecret(t *testing.T) {
	issuer := testDir(t)
	other, err := NewDirectory("other-secret", []User{{Account: "achi", Password: "pw"}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0)
	token, err := issuer.Issue("achi", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Verify(token, now); err != ErrInvalidToken {
		t.Fatalf("other secret: %v", err)
	}
}

func TestPasswordChangeKeepsTokenUntilExpiry(t *testing.T) {
	issuer := testDir(t)
	now := time.Unix(1_700_000_000, 0)
	token, err := issuer.Issue("achi", now)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := NewDirectory("top-secret", []User{{Account: "achi", Password: "new"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := changed.Verify(token, now); err != nil {
		t.Fatalf("password change should not revoke token: %v", err)
	}
}

func TestRemovedAccountRejectsExistingToken(t *testing.T) {
	issuer := testDir(t)
	now := time.Unix(1_700_000_000, 0)
	token, err := issuer.Issue("achi", now)
	if err != nil {
		t.Fatal(err)
	}
	remaining, err := NewDirectory("top-secret", []User{{Account: "other", Password: "pw"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := remaining.Verify(token, now); err != ErrInvalidToken {
		t.Fatalf("removed account: %v", err)
	}
}
