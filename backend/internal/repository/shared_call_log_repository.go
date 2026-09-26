package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/assethub/assethub/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SharedCallLogRepository provides data access for shared credential call logs.
type SharedCallLogRepository struct {
	coll *mongo.Collection
}

// NewSharedCallLogRepository creates a SharedCallLogRepository.
func NewSharedCallLogRepository(db *mongo.Database) *SharedCallLogRepository {
	return &SharedCallLogRepository{coll: db.Collection("shared_call_logs")}
}

// EnsureIndexes creates lookup indexes used when reviewing call logs.
func (r *SharedCallLogRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "credential_id", Value: 1}, {Key: "called_at", Value: -1}},
		Options: options.Index().SetName("idx_credential_called_at"),
	})
	if err != nil {
		return fmt.Errorf("ensure shared call log indexes: %w", err)
	}
	return nil
}

// Create inserts a shared call log entry.
func (r *SharedCallLogRepository) Create(ctx context.Context, logEntry *model.SharedCallLog) error {
	if logEntry.CalledAt.IsZero() {
		logEntry.CalledAt = time.Now()
	}
	res, err := r.coll.InsertOne(ctx, logEntry)
	if err != nil {
		return fmt.Errorf("create shared call log: %w", err)
	}
	logEntry.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

// ListByCredential returns paginated call logs for a credential, newest first.
func (r *SharedCallLogRepository) ListByCredential(ctx context.Context, credentialID primitive.ObjectID, skip, limit int64) ([]model.SharedCallLog, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.D{{Key: "called_at", Value: -1}})
	cursor, err := r.coll.Find(ctx, bson.M{"credential_id": credentialID}, opts)
	if err != nil {
		return nil, fmt.Errorf("list shared call logs: %w", err)
	}
	defer cursor.Close(ctx)
	var logs []model.SharedCallLog
	if err != nil {
		return nil, fmt.Errorf("decode shared call logs: %w", err)
	}
	return logs, nil
}
