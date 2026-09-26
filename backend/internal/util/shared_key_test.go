package util

import (
	"strings"
	"testing"
)

func TestGenerateSharedKey(t *testing.T) {
	t.Parallel()
	key, err := GenerateSharedKey()
	if err != nil {
		t.Fatalf("GenerateSharedKey() error = %v", err)
	}
	if !strings.HasPrefix(key, sharedKeyPrefix) {
		t.Fatalf("key = %q, want prefix %q", key, sharedKeyPrefix)
	}
	// 24 random bytes -> 48 hex chars plus the prefix.
	if len(key) != len(sharedKeyPrefix)+sharedKeyBytes*2 {
		t.Fatalf("len(key) = %d, want %d", len(key), len(sharedKeyPrefix)+sharedKeyBytes*2)
	}

	other, err := GenerateSharedKey()
	if err != nil {
		t.Fatalf("GenerateSharedKey() error = %v", err)
	}
	if key == other {
		t.Fatal("generated keys must be unique")
	}
}

func TestHashSharedKey(t *testing.T) {
	t.Parallel()
	if HashSharedKey("a") == HashSharedKey("b") {
		t.Fatal("distinct keys must hash differently")
	}
	if HashSharedKey("same") != HashSharedKey("same") {
		t.Fatal("hashing must be deterministic")
	}
	if HashSharedKey("ahk_secret") == "ahk_secret" {
		t.Fatal("hash must not equal the plaintext key")
	}
}

func TestSharedKeyLast4(t *testing.T) {
	t.Parallel()
	if got := SharedKeyLast4("ahk_0123456789abcdef"); got != "cdef" {
		t.Fatalf("SharedKeyLast4() = %q, want %q", got, "cdef")
	}
}
