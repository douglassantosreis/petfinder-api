package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/yourname/go-backend/internal/domain/upload"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UploadRepository struct {
	collection *mongo.Collection
}

func NewUploadRepository(db *mongo.Database) *UploadRepository {
	return &UploadRepository{collection: db.Collection("uploads")}
}

func (r *UploadRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
		{Keys: bson.D{{Key: "url", Value: 1}, {Key: "userId", Value: 1}}},
		{Keys: bson.D{{Key: "reportId", Value: 1}}},
		{Keys: bson.D{{Key: "moderationStatus", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: 1}}},
	})
	return err
}

func (r *UploadRepository) Create(ctx context.Context, u domain.Upload) error {
	_, err := r.collection.InsertOne(ctx, u)
	return err
}

func (r *UploadRepository) UpdateModerationStatus(ctx context.Context, uploadID string, status domain.ModerationStatus, reason string) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": uploadID},
		bson.M{"$set": bson.M{"moderationStatus": status, "moderationReason": reason}},
	)
	return err
}

func (r *UploadRepository) FindByReport(ctx context.Context, reportID string) ([]domain.Upload, error) {
	cur, err := r.collection.Find(ctx, bson.M{"reportId": reportID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := make([]domain.Upload, 0)
	for cur.Next(ctx) {
		var u domain.Upload
		if err := cur.Decode(&u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, cur.Err()
}

func (r *UploadRepository) ValidateOwnership(ctx context.Context, urls []string, userID, currentReportID string) error {
	filter := bson.M{
		"url":    bson.M{"$in": urls},
		"userId": userID,
		"$or": bson.A{
			bson.M{"reportId": bson.M{"$exists": false}},
			bson.M{"reportId": ""},
			bson.M{"reportId": currentReportID},
		},
	}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return err
	}
	if int(count) != len(urls) {
		return errors.New("one or more photos are invalid or do not belong to you")
	}
	return nil
}

func (r *UploadRepository) AssignToReport(ctx context.Context, urls []string, reportID string) error {
	_, err := r.collection.UpdateMany(ctx,
		bson.M{"url": bson.M{"$in": urls}},
		bson.M{"$set": bson.M{"reportId": reportID}},
	)
	return err
}

// AllApproved returns true when every URL in urls has moderationStatus "approved".
func (r *UploadRepository) AllApproved(ctx context.Context, urls []string) (bool, error) {
	pending, err := r.collection.CountDocuments(ctx, bson.M{
		"url":              bson.M{"$in": urls},
		"moderationStatus": bson.M{"$ne": string(domain.ModerationApproved)},
	})
	if err != nil {
		return false, err
	}
	return pending == 0, nil
}

func (r *UploadRepository) FindOrphansOlderThan(ctx context.Context, age time.Duration) ([]domain.Upload, error) {
	cutoff := time.Now().UTC().Add(-age)
	filter := bson.M{
		"$or": bson.A{
			bson.M{"reportId": bson.M{"$exists": false}},
			bson.M{"reportId": ""},
		},
		"createdAt": bson.M{"$lt": cutoff},
	}
	cur, err := r.collection.Find(ctx, filter, options.Find().SetLimit(200))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := make([]domain.Upload, 0)
	for cur.Next(ctx) {
		var u domain.Upload
		if err := cur.Decode(&u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, cur.Err()
}

func (r *UploadRepository) Delete(ctx context.Context, id string) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("upload %s not found", id)
	}
	return nil
}
