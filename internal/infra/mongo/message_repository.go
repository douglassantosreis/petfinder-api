package mongo

import (
	"context"
	"time"

	domain "github.com/yourname/go-backend/internal/domain/message"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConversationRepository struct {
	collection *mongo.Collection
}

type MessageRepository struct {
	collection *mongo.Collection
}

func NewConversationRepository(db *mongo.Database) *ConversationRepository {
	return &ConversationRepository{collection: db.Collection("conversations")}
}

func NewMessageRepository(db *mongo.Database) *MessageRepository {
	return &MessageRepository{collection: db.Collection("messages")}
}

func (r *ConversationRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "reportId", Value: 1}}},
		{Keys: bson.D{{Key: "participants", Value: 1}}},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *MessageRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "conversationId", Value: 1}, {Key: "createdAt", Value: 1}}},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *ConversationRepository) Create(ctx context.Context, c domain.Conversation) (domain.Conversation, error) {
	_, err := r.collection.InsertOne(ctx, c)
	return c, err
}

func (r *ConversationRepository) GetByID(ctx context.Context, id string) (domain.Conversation, error) {
	var out domain.Conversation
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&out)
	return out, err
}

func (r *ConversationRepository) ListByUser(ctx context.Context, userID string) ([]domain.Conversation, error) {
	cur, err := r.collection.Find(ctx, bson.M{"participants": userID}, options.Find().SetSort(bson.M{"lastMessageAt": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := make([]domain.Conversation, 0)
	for cur.Next(ctx) {
		var item domain.Conversation
		if err := cur.Decode(&item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, cur.Err()
}

func (r *ConversationRepository) UpdateLastMessageAt(ctx context.Context, id string, at time.Time) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"lastMessageAt": at}})
	return err
}

func (r *ConversationRepository) FindByReportAndRequester(ctx context.Context, reportID, requesterID string) (domain.Conversation, bool, error) {
	filter := bson.M{"reportId": reportID, "participants": requesterID}
	var out domain.Conversation
	err := r.collection.FindOne(ctx, filter).Decode(&out)
	if err == mongo.ErrNoDocuments {
		return domain.Conversation{}, false, nil
	}
	if err != nil {
		return domain.Conversation{}, false, err
	}
	return out, true, nil
}

func (r *MessageRepository) Create(ctx context.Context, m domain.Message) (domain.Message, error) {
	_, err := r.collection.InsertOne(ctx, m)
	return m, err
}

func (r *MessageRepository) ListByConversation(ctx context.Context, conversationID string, page, pageSize int) ([]domain.Message, error) {
	skip := int64((page - 1) * pageSize)
	opts := options.Find().
		SetSort(bson.M{"createdAt": 1}).
		SetSkip(skip).
		SetLimit(int64(pageSize))

	cur, err := r.collection.Find(ctx, bson.M{"conversationId": conversationID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := make([]domain.Message, 0, pageSize)
	for cur.Next(ctx) {
		var item domain.Message
		if err := cur.Decode(&item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, cur.Err()
}
