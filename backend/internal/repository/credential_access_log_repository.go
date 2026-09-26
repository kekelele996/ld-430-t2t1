package repository

import (
	"context"
	"fmt"

	"github.com/assethub/assethub/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CredentialAccessLogRepository provides data access for credential access logs.
type CredentialAccessLogRepository struct {
	coll *mongo.Collection
}

// NewCredentialAccessLogRepository creates a CredentialAccessLogRepository.
func NewCredentialAccessLogRepository(db *mongo.Database) *CredentialAccessLogRepository {
	return &CredentialAccessLogRepository{coll: db.Collection("credential_access_logs")}
}

// Create inserts a credential access log entry.
func (r *CredentialAccessLogRepository) Create(ctx context.Context, entry *model.CredentialAccessLog) error {
	res, err := r.coll.InsertOne(ctx, entry)
	if err != nil {
		return fmt.Errorf("create credential access log: %w", err)
	}
	entry.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

// ListByCredential returns access logs of a credential with pagination.
func (r *CredentialAccessLogRepository) ListByCredential(ctx context.Context, credentialID primitive.ObjectID, skip, limit int64) ([]model.CredentialAccessLog, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.coll.Find(ctx, bson.M{"credential_id": credentialID}, opts)
	if err != nil {
		return nil, fmt.Errorf("list credential access logs: %w", err)
	}
	defer cursor.Close(ctx)
	var entries []model.CredentialAccessLog
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("decode credential access logs: %w", err)
	}
	return entries, nil
}
