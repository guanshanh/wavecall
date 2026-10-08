package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

// User is one hand-edited account. Password is stored as written.
type User struct {
	Account  string `toml:"account"`
	Password string `toml:"password"`
}

// Directory is the user roster loaded once at process start.
type Directory struct {
	secret    string
	passwords map[string]string
}

type fileDoc struct {
	Secret string `toml:"secret"`
	Users  []User `toml:"users"`
}

// Load reads users.toml and validates it.
func Load(path string) (*Directory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open users file: %w", err)
	}
	var doc fileDoc
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse users file: %w", err)
	}
	return NewDirectory(doc.Secret, doc.Users)
}

// NewDirectory validates secret and users. Account is trimmed; password is not.
func NewDirectory(secret string, users []User) (*Directory, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("empty secret")
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("no users")
	}
	passwords := make(map[string]string, len(users))
	for _, u := range users {
		account := strings.TrimSpace(u.Account)
		if !validAccount(account) {
			return nil, fmt.Errorf("invalid account %q", u.Account)
		}
		if u.Password == "" {
			return nil, fmt.Errorf("empty password for %q", account)
		}
		if _, exists := passwords[account]; exists {
			return nil, fmt.Errorf("duplicate account %q", account)
		}
		passwords[account] = u.Password
	}
	return &Directory{secret: secret, passwords: passwords}, nil
}

func validAccount(account string) bool {
	if account == "" || strings.Contains(account, "|") {
		return false
	}
	for _, r := range account {
		if unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// Authenticate checks the password. The bool is false for unknown accounts and
// wrong passwords. On success, canonical is the roster spelling.
func (d *Directory) Authenticate(account, password string) (string, bool) {
	stored, ok := d.passwords[account]
	if !ok {
		stored = ""
	}
	sumIn := sha256.Sum256([]byte(password))
	sumStored := sha256.Sum256([]byte(stored))
	match := subtle.ConstantTimeCompare(sumIn[:], sumStored[:]) == 1
	if !ok || !match {
		return "", false
	}
	return account, true
}
