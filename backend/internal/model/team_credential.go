package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TeamCredential is a shared, team-level API credential issued to external services.
// The plaintext secret is shown exactly once at creation; only its hash and the
// last four characters are persisted.
type TeamCredential struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name       string             `bson:"name" json:"name"`
	KeyID      string             `bson:"key_id" json:"key_id"`
	SecretHash string             `bson:"secret_hash" json:"-"`
	LastFour   string             `bson:"last_four" json:"last_four"`
	Scopes     []string           `bson:"scopes" json:"scopes"`
	DailyLimit int                `bson:"daily_limit" json:"daily_limit"`
	Status     string             `bson:"status" json:"status"`
	ExpiresAt  *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	CreatedBy  primitive.ObjectID `bson:"created_by" json:"created_by"`
	RevokedBy  primitive.ObjectID `bson:"revoked_by,omitempty" json:"revoked_by,omitempty"`
	RevokedAt  *time.Time         `bson:"revoked_at,omitempty" json:"revoked_at,omitempty"`
	LastUsedAt *time.Time         `bson:"last_used_at,omitempty" json:"last_used_at,omitempty"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}
