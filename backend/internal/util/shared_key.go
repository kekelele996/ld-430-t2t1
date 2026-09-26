package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// sharedKeyPrefix is the public prefix of every team shared credential key.
const sharedKeyPrefix = "ahk_"

// sharedKeyBytes is the entropy (in bytes) behind the random portion of a key.
const sharedKeyBytes = 24

// GenerateSharedKey generates a new plaintext shared credential key.
func GenerateSharedKey() (string, error) {
	buf := make([]byte, sharedKeyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate shared key: %w", err)
	}
	return sharedKeyPrefix + hex.EncodeToString(buf), nil
}

// HashSharedKey returns the hex-encoded SHA-256 digest of a plaintext key.
func HashSharedKey(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// SharedKeyLast4 returns the last four characters of a plaintext key.
func SharedKeyLast4(plain string) string {
	if len(plain) < 4 {
		return plain
	}
	return plain[len(plain)-4:]
}
