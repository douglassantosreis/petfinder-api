package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type revokedToken struct {
	JTI       string    `bson:"_id"`
	ExpiresAt time.Time `bson:"expiresAt"`
}

type TokenRevocationRepository struct {
	collection *mongo.Collection
}

func NewTokenRevocationRepository(db *mongo.Database) *TokenRevocationRepository {
	return &TokenRevocationRepository{collection: db.Collection("revoked_tokens")}
}

// EnsureIndexes creates a TTL index so MongoDB auto-purges expired tokens.
func (r *TokenRevocationRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	return err
}

func (r *TokenRevocationRepository) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
	_, err := r.collection.InsertOne(ctx, revokedToken{JTI: jti, ExpiresAt: expiresAt})
	// ignore duplicate key — already revoked is fine
	if mongo.IsDuplicateKeyError(err) {
		return nil
	}
	return err
}

func (r *TokenRevocationRepository) IsRevoked(ctx context.Context, jti string) (bool, error) {
	err := r.collection.FindOne(ctx, bson.M{"_id": jti}).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
