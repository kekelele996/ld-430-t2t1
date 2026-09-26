package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SharedCredential is a team-scoped API credential issued to external services.
// The plaintext key is shown exactly once at creation; only its SHA-256 hash and
// the last four characters are persisted.
type SharedCredential struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name       string             `bson:"name" json:"name"`
	KeyHash    string             `bson:"key_hash" json:"-"`
	KeyLast4   string             `bson:"key_last4" json:"key_last4"`
	Scopes     []string           `bson:"scopes" json:"scopes"`
	DailyLimit int64              `bson:"daily_limit" json:"daily_limit"`
	ExpiresAt  *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	Revoked    bool               `bson:"revoked" json:"revoked"`
	CreatedBy  primitive.ObjectID `bson:"created_by" json:"created_by"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
	RevokedAt  *time.Time         `bson:"revoked_at,omitempty" json:"revoked_at,omitempty"`
}
