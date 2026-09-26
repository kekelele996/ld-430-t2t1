package dto

import "time"

// CreateSharedCredentialRequest is the admin payload for issuing a shared credential.
type CreateSharedCredentialRequest struct {
	Name       string     `json:"name" validate:"required,min=1,max=100"`
	Scopes     []string   `json:"scopes" validate:"required,min=1,dive,oneof=assets:read collections:write downloads:write"`
	DailyLimit int64      `json:"daily_limit" validate:"required,gt=0"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// SharedCredentialCreated is returned exactly once and embeds the plaintext key.
type SharedCredentialCreated struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Scopes     []string   `json:"scopes"`
	DailyLimit int64      `json:"daily_limit"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Key        string     `json:"key"`
	KeyLast4   string     `json:"key_last4"`
	CreatedAt  time.Time  `json:"created_at"`
}

// SharedCredentialItem is the safe list projection: name, scopes, quota, last four.
type SharedCredentialItem struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Scopes     []string   `json:"scopes"`
	DailyLimit int64      `json:"daily_limit"`
	KeyLast4   string     `json:"key_last4"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Revoked    bool       `json:"revoked"`
	CreatedAt  time.Time  `json:"created_at"`
}
