package mongo

import (
	"context"

	domain "github.com/yourname/go-backend/internal/domain/ad"
	aduc "github.com/yourname/go-backend/internal/usecase/ad"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AdRepository struct {
	collection *mongo.Collection
}

func NewAdRepository(db *mongo.Database) *AdRepository {
	return &AdRepository{collection: db.Collection("reports")}
}

func (r *AdRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "visible", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "lastSeenLocation", Value: "2dsphere"}}},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *AdRepository) Create(ctx context.Context, report domain.FoundAnimalReport) (domain.FoundAnimalReport, error) {
	_, err := r.collection.InsertOne(ctx, report)
	return report, err
}

func (r *AdRepository) GetByID(ctx context.Context, id string) (domain.FoundAnimalReport, error) {
	var out domain.FoundAnimalReport
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&out)
	return out, err
}

func (r *AdRepository) SetVisible(ctx context.Context, reportID string, visible bool) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": reportID},
		bson.M{"$set": bson.M{"visible": visible}},
	)
	return err
}

// ListOpen returns open visible reports.
// visible:{$ne:false} matches true AND missing field (backward-compatible with old documents).
func (r *AdRepository) ListOpen(ctx context.Context, page, pageSize int, geo *aduc.GeoFilter) ([]domain.FoundAnimalReport, error) {
	skip := int64((page - 1) * pageSize)

	baseFilter := bson.M{
		"status":  domain.StatusOpen,
		"visible": bson.M{"$ne": false},
	}

	var filter bson.M
	var opts *options.FindOptions

	if geo != nil {
		filter = bson.M{
			"status":  baseFilter["status"],
			"visible": baseFilter["visible"],
			"lastSeenLocation": bson.M{
				"$near": bson.M{
					"$geometry": bson.M{
						"type":        "Point",
						"coordinates": []float64{geo.Longitude, geo.Latitude},
					},
					"$maxDistance": geo.RadiusMeters(),
				},
			},
		}
		opts = options.Find().SetSkip(skip).SetLimit(int64(pageSize))
	} else {
		filter = baseFilter
		opts = options.Find().
			SetSort(bson.M{"createdAt": -1}).
			SetSkip(skip).
			SetLimit(int64(pageSize))
	}

	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := make([]domain.FoundAnimalReport, 0, pageSize)
	for cur.Next(ctx) {
		var item domain.FoundAnimalReport
		if err := cur.Decode(&item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, cur.Err()
}

func (r *AdRepository) Update(ctx context.Context, report domain.FoundAnimalReport) (domain.FoundAnimalReport, error) {
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": report.ID}, report)
	return report, err
}
