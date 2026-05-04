package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/yourname/go-backend/internal/domain/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{collection: db.Collection("users")}
}

func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	// Drop the legacy full unique index on (oauthProvider, oauthSubject) if it
	// still exists. The replacement has a partial filter so it only applies to
	// OAuth users, allowing multiple email/password users with empty provider.
	_, _ = r.collection.Indexes().DropOne(ctx, "oauthProvider_1_oauthSubject_1")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "oauthProvider", Value: 1}, {Key: "oauthSubject", Value: 1}},
			Options: options.Index().
				SetName("idx_oauth_provider_subject_unique").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"oauthProvider": bson.M{"$gt": ""}}),
		},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *UserRepository) UpsertByOAuth(ctx context.Context, input domain.User) (domain.User, error) {
	filter := bson.M{
		"oauthProvider": input.OAuthProvider,
		"oauthSubject":  input.OAuthSubject,
	}
	update := bson.M{
		"$set": bson.M{
			"name":      input.Name,
			"email":     input.Email,
			"avatarUrl": input.AvatarURL,
			"status":    domain.StatusActive,
			"updatedAt": input.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"_id":           input.ID,
			"oauthProvider": input.OAuthProvider,
			"oauthSubject":  input.OAuthSubject,
			"createdAt":     input.CreatedAt,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var out domain.User
	if err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&out); err != nil {
		return domain.User{}, err
	}
	return out, nil
}

func (r *UserRepository) CreateWithPassword(ctx context.Context, u domain.User) (domain.User, error) {
	_, err := r.collection.InsertOne(ctx, u)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.User{}, fmt.Errorf("%w", domain.ErrEmailTaken)
		}
		return domain.User{}, err
	}
	return u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	var out domain.User
	err := r.collection.FindOne(ctx, bson.M{
		"email":  email,
		"status": bson.M{"$ne": domain.StatusInactive},
	}).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.User{}, domain.ErrNotFound
	}
	return out, err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	var out domain.User
	err := r.collection.FindOne(ctx, bson.M{
		"_id":    id,
		"status": bson.M{"$ne": domain.StatusInactive},
	}).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.User{}, domain.ErrNotFound
	}
	return out, err
}

func (r *UserRepository) UpdateMe(ctx context.Context, id string, name string, city string, state string) (domain.User, error) {
	update := bson.M{
		"$set": bson.M{
			"name":      name,
			"city":      city,
			"state":     state,
			"updatedAt": time.Now().UTC(),
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var out domain.User
	if err := r.collection.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&out); err != nil {
		return domain.User{}, err
	}
	return out, nil
}

func (r *UserRepository) SoftDelete(ctx context.Context, id string, at time.Time) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"status":    domain.StatusInactive,
			"updatedAt": at,
		},
	})
	return err
}
