package dto

import "time"

// CreateTeamCredentialRequest is the payload for issuing a team credential.
type CreateTeamCredentialRequest struct {
	Name       string   `json:"name" validate:"required,min=1,max=100"`
	Scopes     []string `json:"scopes" validate:"required,min=1,dive,oneof=assets:read collections:write downloads:write"`
	DailyLimit int      `json:"daily_limit" validate:"required,min=1,max=1000000"`
	ExpiresAt  string   `json:"expires_at" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

// RevokeTeamCredentialRequest revokes a credential.
type RevokeTeamCredentialRequest struct {
	Reason string `json:"reason"`
}

// TeamCredentialCreatedResponse is returned exactly once when a credential is
// created. The plaintext api_key is never shown again.
type TeamCredentialCreatedResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	APIKey     string    `json:"api_key"`
	Scopes     []string  `json:"scopes"`
	DailyLimit int       `json:"daily_limit"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// TeamCredentialSummary is the list item; only name, scopes, daily quota and the
// key's last four characters are exposed (no full key, status or timestamps).
type TeamCredentialSummary struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Scopes     []string `json:"scopes"`
	DailyLimit int      `json:"daily_limit"`
	LastFour   string   `json:"last_four"`
}
