package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// TokenTTL is how long a login token stays valid.
const TokenTTL = 12 * time.Hour

// ErrInvalidToken is returned for every verify failure.
var ErrInvalidToken = errors.New("invalid token")

type tokenPayload struct {
	Account string `json:"account"`
	Exp     int64  `json:"exp"`
}

// Issue signs a token for an account that is currently in the roster.
func (d *Directory) Issue(account string, now time.Time) (string, error) {
	if _, ok := d.passwords[account]; !ok {
		return "", ErrInvalidToken
	}
	raw, err := json.Marshal(tokenPayload{
		Account: account,
		Exp:     now.Add(TokenTTL).Unix(),
	})
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(d.secret))
	_, _ = mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// Verify checks signature, expiry, and that the account is still on the roster.
// Valid when now.Unix() <= exp.
func (d *Directory) Verify(token string, now time.Time) (string, error) {
	left, right, ok := strings.Cut(token, ".")
	if !ok || left == "" || right == "" || strings.Contains(right, ".") {
		return "", ErrInvalidToken
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(left)
	if err != nil {
		return "", ErrInvalidToken
	}
	macBytes, err := base64.RawURLEncoding.DecodeString(right)
	if err != nil {
		return "", ErrInvalidToken
	}
	mac := hmac.New(sha256.New, []byte(d.secret))
	_, _ = mac.Write(payloadBytes)
	if !hmac.Equal(macBytes, mac.Sum(nil)) {
		return "", ErrInvalidToken
	}
	var payload tokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", ErrInvalidToken
	}
	if now.Unix() > payload.Exp {
		return "", ErrInvalidToken
	}
	if _, ok := d.passwords[payload.Account]; !ok {
		return "", ErrInvalidToken
	}
	return payload.Account, nil
}
