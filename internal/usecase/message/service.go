package message

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	addomain "github.com/yourname/go-backend/internal/domain/ad"
	domain "github.com/yourname/go-backend/internal/domain/message"
)

var (
	ErrNotParticipant   = errors.New("user is not a participant")
	ErrSelfConversation = errors.New("cannot start a conversation with yourself")
	ErrReportNotFound   = errors.New("report not found")
)

type ReportRepository interface {
	GetByID(ctx context.Context, id string) (addomain.FoundAnimalReport, error)
}

type ConversationRepository interface {
	Create(ctx context.Context, c domain.Conversation) (domain.Conversation, error)
	GetByID(ctx context.Context, id string) (domain.Conversation, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Conversation, error)
	UpdateLastMessageAt(ctx context.Context, id string, at time.Time) error
	// FindByReportAndRequester returns the existing conversation, if any.
	FindByReportAndRequester(ctx context.Context, reportID, requesterID string) (domain.Conversation, bool, error)
}

type MessageRepository interface {
	Create(ctx context.Context, m domain.Message) (domain.Message, error)
	ListByConversation(ctx context.Context, conversationID string, page, pageSize int) ([]domain.Message, error)
}

type Service struct {
	conversations ConversationRepository
	messages      MessageRepository
	reports       ReportRepository
}

func NewService(conversations ConversationRepository, messages MessageRepository, reports ReportRepository) *Service {
	return &Service{
		conversations: conversations,
		messages:      messages,
		reports:       reports,
	}
}

// StartConversation opens a conversation between the requester and the report owner.
// If a conversation for this report+requester already exists, the existing one is returned (idempotent).
func (s *Service) StartConversation(ctx context.Context, reportID string, requesterID string) (domain.Conversation, error) {
	report, err := s.reports.GetByID(ctx, reportID)
	if err != nil {
		return domain.Conversation{}, ErrReportNotFound
	}
	if report.OwnerID == requesterID {
		return domain.Conversation{}, ErrSelfConversation
	}

	existing, found, err := s.conversations.FindByReportAndRequester(ctx, reportID, requesterID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if found {
		return existing, nil
	}

	now := time.Now().UTC()
	return s.conversations.Create(ctx, domain.Conversation{
		ID:            uuid.NewString(),
		ReportID:      reportID,
		Participants:  []string{requesterID, report.OwnerID},
		CreatedAt:     now,
		LastMessageAt: now,
	})
}

func (s *Service) ListConversations(ctx context.Context, userID string) ([]domain.Conversation, error) {
	return s.conversations.ListByUser(ctx, userID)
}

func (s *Service) ListMessages(ctx context.Context, conversationID string, userID string, page, pageSize int) ([]domain.Message, error) {
	c, err := s.conversations.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(c.Participants, userID) {
		return nil, ErrNotParticipant
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return s.messages.ListByConversation(ctx, conversationID, page, pageSize)
}

func (s *Service) SendMessage(ctx context.Context, conversationID string, senderID string, body string) (domain.Message, error) {
	c, err := s.conversations.GetByID(ctx, conversationID)
	if err != nil {
		return domain.Message{}, err
	}
	if !slices.Contains(c.Participants, senderID) {
		return domain.Message{}, ErrNotParticipant
	}
	now := time.Now().UTC()
	msg, err := s.messages.Create(ctx, domain.Message{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Body:           body,
		CreatedAt:      now,
	})
	if err != nil {
		return domain.Message{}, err
	}
	if err := s.conversations.UpdateLastMessageAt(ctx, conversationID, now); err != nil {
		slog.Warn("update last message at failed", "conversationId", conversationID, "error", err)
	}
	return msg, nil
}
