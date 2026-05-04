package message

import (
	"context"
	"errors"
	"testing"
	"time"

	addomain "github.com/yourname/go-backend/internal/domain/ad"
	domain "github.com/yourname/go-backend/internal/domain/message"
)

// --- fakes ---

type fakeConversationRepo struct {
	item domain.Conversation
}

func (f *fakeConversationRepo) Create(_ context.Context, c domain.Conversation) (domain.Conversation, error) {
	f.item = c
	return c, nil
}
func (f *fakeConversationRepo) GetByID(_ context.Context, _ string) (domain.Conversation, error) {
	return f.item, nil
}
func (f *fakeConversationRepo) ListByUser(_ context.Context, _ string) ([]domain.Conversation, error) {
	return []domain.Conversation{f.item}, nil
}
func (f *fakeConversationRepo) UpdateLastMessageAt(_ context.Context, _ string, at time.Time) error {
	f.item.LastMessageAt = at
	return nil
}
func (f *fakeConversationRepo) FindByReportAndRequester(_ context.Context, _, _ string) (domain.Conversation, bool, error) {
	if f.item.ID != "" {
		return f.item, true, nil
	}
	return domain.Conversation{}, false, nil
}

type fakeEmptyConversationRepo struct {
	fakeConversationRepo
}

func (f *fakeEmptyConversationRepo) FindByReportAndRequester(_ context.Context, _, _ string) (domain.Conversation, bool, error) {
	return domain.Conversation{}, false, nil
}

type fakeMessageRepo struct{}

func (f *fakeMessageRepo) Create(_ context.Context, m domain.Message) (domain.Message, error) {
	return m, nil
}
func (f *fakeMessageRepo) ListByConversation(_ context.Context, _ string, _, _ int) ([]domain.Message, error) {
	return []domain.Message{}, nil
}

type fakeReportRepo struct {
	report addomain.FoundAnimalReport
	err    error
}

func (f *fakeReportRepo) GetByID(_ context.Context, _ string) (addomain.FoundAnimalReport, error) {
	return f.report, f.err
}

// --- tests ---

func TestSendMessageFailsForNonParticipant(t *testing.T) {
	convRepo := &fakeConversationRepo{
		item: domain.Conversation{ID: "c1", Participants: []string{"u1", "u2"}},
	}
	svc := NewService(convRepo, &fakeMessageRepo{}, &fakeReportRepo{})

	_, err := svc.SendMessage(context.Background(), "c1", "u3", "hello")
	if err == nil {
		t.Fatal("expected participant validation error")
	}
}

func TestStartConversationFailsForOwner(t *testing.T) {
	svc := NewService(
		&fakeEmptyConversationRepo{},
		&fakeMessageRepo{},
		&fakeReportRepo{report: addomain.FoundAnimalReport{ID: "r1", OwnerID: "owner-1"}},
	)

	_, err := svc.StartConversation(context.Background(), "r1", "owner-1")
	if !errors.Is(err, ErrSelfConversation) {
		t.Fatalf("expected ErrSelfConversation, got %v", err)
	}
}

func TestStartConversationReturnsExistingWhenDuplicate(t *testing.T) {
	existing := domain.Conversation{ID: "existing-conv", ReportID: "r1", Participants: []string{"finder-1", "owner-1"}}
	convRepo := &fakeConversationRepo{item: existing}
	svc := NewService(
		convRepo,
		&fakeMessageRepo{},
		&fakeReportRepo{report: addomain.FoundAnimalReport{ID: "r1", OwnerID: "owner-1"}},
	)

	got, err := svc.StartConversation(context.Background(), "r1", "finder-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != existing.ID {
		t.Fatalf("expected existing conversation %q, got %q", existing.ID, got.ID)
	}
}

func TestStartConversationFailsWhenReportNotFound(t *testing.T) {
	svc := NewService(
		&fakeEmptyConversationRepo{},
		&fakeMessageRepo{},
		&fakeReportRepo{err: errors.New("not found")},
	)

	_, err := svc.StartConversation(context.Background(), "r-missing", "finder-1")
	if !errors.Is(err, ErrReportNotFound) {
		t.Fatalf("expected ErrReportNotFound, got %v", err)
	}
}
