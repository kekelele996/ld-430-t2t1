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

// SharedCredentialRepository provides data access for team shared credentials.
type SharedCredentialRepository struct {
	coll *mongo.Collection
}

// NewSharedCredentialRepository creates a SharedCredentialRepository.
func NewSharedCredentialRepository(db *mongo.Database) *SharedCredentialRepository {
	return &SharedCredentialRepository{coll: db.Collection("shared_credentials")}
}

// EnsureIndexes creates lookup indexes used during shared credential authentication.
func (r *SharedCredentialRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key_hash", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_key_hash"),
		},
	})
	if err != nil {
		return fmt.Errorf("ensure shared credential indexes: %w", err)
	}
	return nil
}

// Create inserts a shared credential.
func (r *SharedCredentialRepository) Create(ctx context.Context, credential *model.SharedCredential) error {
	now := time.Now()
	credential.CreatedAt = now
	credential.UpdatedAt = now
	res, err := r.coll.InsertOne(ctx, credential)
	if err != nil {
		return fmt.Errorf("create shared credential: %w", err)
	}
	credential.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByHash returns a shared credential by its key hash.
func (r *SharedCredentialRepository) FindByHash(ctx context.Context, keyHash string) (*model.SharedCredential, error) {
	var credential model.SharedCredential
	err := r.coll.FindOne(ctx, bson.M{"key_hash": keyHash}).Decode(&credential)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find shared credential by hash: %w", err)
	}
	return &credential, nil
}

// FindByID returns a shared credential by id.
func (r *SharedCredentialRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.SharedCredential, error) {
	var credential model.SharedCredential
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&credential)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find shared credential by id: %w", err)
	}
	return &credential, nil
}

// List returns all shared credentials sorted by creation time, newest first.
func (r *SharedCredentialRepository) List(ctx context.Context) ([]model.SharedCredential, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("list shared credentials: %w", err)
	}
	defer cursor.Close(ctx)
	var credentials []model.SharedCredential
	if err := cursor.All(ctx, &credentials); err != nil {
		return nil, fmt.Errorf("decode shared credentials: %w", err)
	}
	return credentials, nil
}

// Revoke marks a shared credential as revoked.
func (r *SharedCredentialRepository) Revoke(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	res, err := r.coll.UpdateOne(ctx,
		bson.M{"_id": id, "revoked": false},
		bson.M{"$set": bson.M{"revoked": true, "revoked_at": now, "updated_at": now}},
	)
	if err != nil {
		return fmt.Errorf("revoke shared credential: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}
