package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TeamCredentialRepository provides data access for team credentials.
type TeamCredentialRepository struct {
	coll *mongo.Collection
}

// NewTeamCredentialRepository creates a TeamCredentialRepository.
func NewTeamCredentialRepository(db *mongo.Database) *TeamCredentialRepository {
	return &TeamCredentialRepository{coll: db.Collection("team_credentials")}
}

// Create inserts a team credential.
func (r *TeamCredentialRepository) Create(ctx context.Context, credential *model.TeamCredential) error {
	now := time.Now()
	credential.CreatedAt = now
	credential.UpdatedAt = now
	res, err := r.coll.InsertOne(ctx, credential)
	if err != nil {
		return fmt.Errorf("create team credential: %w", err)
	}
	credential.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByKeyID returns a team credential by its public lookup key id.
func (r *TeamCredentialRepository) FindByKeyID(ctx context.Context, keyID string) (*model.TeamCredential, error) {
	var credential model.TeamCredential
	err := r.coll.FindOne(ctx, bson.M{"key_id": keyID}).Decode(&credential)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find team credential by key id: %w", err)
	}
	return &credential, nil
}

// FindByID returns a team credential by its object id.
func (r *TeamCredentialRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.TeamCredential, error) {
	var credential model.TeamCredential
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&credential)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find team credential by id: %w", err)
	}
	return &credential, nil
}

// List returns team credentials ordered by creation time descending.
func (r *TeamCredentialRepository) List(ctx context.Context, skip, limit int64) ([]model.TeamCredential, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("list team credentials: %w", err)
	}
	defer cursor.Close(ctx)
	var credentials []model.TeamCredential
	if err := cursor.All(ctx, &credentials); err != nil {
		return nil, fmt.Errorf("decode team credentials: %w", err)
	}
	return credentials, nil
}

// Revoke marks a credential revoked.
func (r *TeamCredentialRepository) Revoke(ctx context.Context, id, revokedBy primitive.ObjectID, revokedAt time.Time) error {
	res, err := r.coll.UpdateOne(ctx,
		bson.M{"_id": id, "status": string(constants.CredentialStatusActive)},
		bson.M{"$set": bson.M{
			"status":     "revoked",
			"revoked_by": revokedBy,
			"revoked_at": revokedAt,
			"updated_at": revokedAt,
		}},
	)
	if err != nil {
		return fmt.Errorf("revoke team credential: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchLastUsed records the last successful usage time.
func (r *TeamCredentialRepository) TouchLastUsed(ctx context.Context, id primitive.ObjectID, usedAt time.Time) error {
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"last_used_at": usedAt}})
	if err != nil {
		return fmt.Errorf("touch team credential last used: %w", err)
	}
	return nil
}
