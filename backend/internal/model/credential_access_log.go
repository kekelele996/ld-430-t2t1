package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CredentialAccessLog records every authenticated call made with a team credential
// through the shared external interface.
type CredentialAccessLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CredentialID primitive.ObjectID `bson:"credential_id,omitempty" json:"credential_id,omitempty"`
	KeyID        string             `bson:"key_id" json:"key_id"`
	Method       string             `bson:"method" json:"method"`
	Path         string             `bson:"path" json:"path"`
	StatusCode   int                `bson:"status_code" json:"status_code"`
	Result       string             `bson:"result" json:"result"`
	IP           string             `bson:"ip" json:"ip"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}
