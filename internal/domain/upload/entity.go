package upload

import "time"

type ModerationStatus string

const (
	ModerationPending  ModerationStatus = "pending"
	ModerationApproved ModerationStatus = "approved"
	ModerationRejected ModerationStatus = "rejected"
)

type Upload struct {
	ID               string           `bson:"_id"`
	UserID           string           `bson:"userId"`
	Filename         string           `bson:"filename"`
	URL              string           `bson:"url"`
	ReportID         string           `bson:"reportId,omitempty"`
	ModerationStatus ModerationStatus `bson:"moderationStatus"`
	ModerationReason string           `bson:"moderationReason,omitempty"`
	CreatedAt        time.Time        `bson:"createdAt"`
}
