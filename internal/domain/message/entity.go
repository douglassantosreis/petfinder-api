package message

import "time"

type Conversation struct {
	ID            string    `bson:"_id" json:"id"`
	ReportID      string    `bson:"reportId" json:"reportId"`
	Participants  []string  `bson:"participants" json:"participants"`
	CreatedAt     time.Time `bson:"createdAt" json:"createdAt"`
	LastMessageAt time.Time `bson:"lastMessageAt" json:"lastMessageAt"`
}

type Message struct {
	ID             string    `bson:"_id" json:"id"`
	ConversationID string    `bson:"conversationId" json:"conversationId"`
	SenderID       string    `bson:"senderId" json:"senderId"`
	Body           string    `bson:"body" json:"body"`
	CreatedAt      time.Time `bson:"createdAt" json:"createdAt"`
}
