package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SharedCallLog records every authenticated call made with a team shared credential,
// including calls denied for expiry/revocation, missing scope or exhausted quota.
type SharedCallLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CredentialID primitive.ObjectID `bson:"credential_id" json:"credential_id"`
	Method       string             `bson:"method" json:"method"`
	Path         string             `bson:"path" json:"path"`
	StatusCode   int                `bson:"status_code" json:"status_code"`
	Result       string             `bson:"result" json:"result"`
	IP           string             `bson:"ip" json:"ip"`
	CalledAt     time.Time          `bson:"called_at" json:"called_at"`
}
