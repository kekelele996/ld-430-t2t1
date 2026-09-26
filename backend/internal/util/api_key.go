package util

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// APIKeyPrefix is the human-readable prefix of every team credential key.
	APIKeyPrefix = "ahk_"
	// apiKeyIDBytes is the entropy (bytes) of the lookup key id.
	apiKeyIDBytes = 12
	// apiKeySecretBytes is the entropy (bytes) of the secret portion.
	apiKeySecretBytes = 24
)

// APIKey is a freshly generated team credential key.
type APIKey struct {
	KeyID    string
	Secret   string
	Full     string
	LastFour string
}

// GenerateAPIKey creates a new random API key. Callers receive the full key
// exactly once; only its hash and last four characters are persisted.
func GenerateAPIKey() (*APIKey, error) {
	id := make([]byte, apiKeyIDBytes)
	if _, err := rand.Read(id); err != nil {
		return nil, fmt.Errorf("generate key id: %w", err)
	}
	secret := make([]byte, apiKeySecretBytes)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate key secret: %w", err)
	}
	keyID := hex.EncodeToString(id)
	secretHex := hex.EncodeToString(secret)
	full := APIKeyPrefix + keyID + secretHex
	return &APIKey{
		KeyID:    keyID,
		Secret:   secretHex,
		Full:     full,
		LastFour: full[len(full)-4:],
	}, nil
}

// ParseAPIKey splits a presented key into its lookup id and secret.
func ParseAPIKey(full string) (keyID, secret string, ok bool) {
	if !strings.HasPrefix(full, APIKeyPrefix) {
		return "", "", false
	}
	body := strings.TrimPrefix(full, APIKeyPrefix)
	wantLen := apiKeyIDBytes*2 + apiKeySecretBytes*2
	if len(body) != wantLen {
		return "", "", false
	}
	return body[:apiKeyIDBytes*2], body[apiKeyIDBytes*2:], true
}

// HashAPIKeySecret returns the SHA-256 hex digest stored for verification.
func HashAPIKeySecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// VerifyAPIKeySecret compares a presented secret against the stored hash in
// constant time.
func VerifyAPIKeySecret(secret, hash string) bool {
	present := HashAPIKeySecret(secret)
	return subtle.ConstantTimeCompare([]byte(present), []byte(hash)) == 1
}
