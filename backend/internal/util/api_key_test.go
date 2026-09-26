package util

import "testing"

func TestGenerateParseVerifyAPIKey(t *testing.T) {
	t.Parallel()

	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	if len(key.Full) != len(APIKeyPrefix)+apiKeyIDBytes*2+apiKeySecretBytes*2 {
		t.Fatalf("unexpected key length %d", len(key.Full))
	}
	if len(key.LastFour) != 4 {
		t.Fatalf("last four length = %d, want 4", len(key.LastFour))
	}
	if key.Full[len(key.Full)-4:] != key.LastFour {
		t.Fatalf("last four %q does not match tail of %q", key.LastFour, key.Full)
	}

	id, secret, ok := ParseAPIKey(key.Full)
	if !ok {
		t.Fatal("ParseAPIKey rejected a generated key")
	}
	if id != key.KeyID || secret != key.Secret {
		t.Fatal("parsed id/secret differ from generated values")
	}
	if !VerifyAPIKeySecret(secret, HashAPIKeySecret(secret)) {
		t.Fatal("VerifyAPIKeySecret rejected the matching secret")
	}
	if VerifyAPIKeySecret(secret+"00", HashAPIKeySecret(secret)) {
		t.Fatal("VerifyAPIKeySecret accepted a tampered secret")
	}
}

func TestParseAPIKeyInvalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
	}{
		{name: "empty", key: ""},
		{name: "wrong prefix", key: "sk_ab12"},
		{name: "too short", key: APIKeyPrefix + "abcd"},
		{name: "too long", key: APIKeyPrefix + "0000000000000000000000000000000000000000000000000000000000000000000000000000"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, _, ok := ParseAPIKey(tt.key); ok {
				t.Fatalf("ParseAPIKey(%q) unexpectedly succeeded", tt.key)
			}
		})
	}
}
