package upload

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	domain "github.com/yourname/go-backend/internal/domain/upload"
)

var (
	ErrFileTooLarge    = errors.New("file exceeds maximum allowed size")
	ErrUnsupportedType = errors.New("unsupported file type; allowed: jpeg, png, webp")
)

var allowedMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type Storage interface {
	Upload(ctx context.Context, filename string, src io.Reader) (url string, err error)
	Delete(ctx context.Context, filename string) error
}

type Repository interface {
	Create(ctx context.Context, u domain.Upload) error
	FindOrphansOlderThan(ctx context.Context, age time.Duration) ([]domain.Upload, error)
	Delete(ctx context.Context, id string) error
}

// ModerationTrigger is implemented by the moderation service.
type ModerationTrigger interface {
	ModerateAsync(upload domain.Upload)
}

type Service struct {
	storage    Storage
	repo       Repository
	moderator  ModerationTrigger
	maxBytes   int64
}

func NewService(storage Storage, repo Repository, moderator ModerationTrigger, maxMB int) *Service {
	return &Service{
		storage:   storage,
		repo:      repo,
		moderator: moderator,
		maxBytes:  int64(maxMB) << 20,
	}
}

func (s *Service) MaxBytes() int64 { return s.maxBytes }

// Upload validates, stores the file, and persists metadata linked to userID.
func (s *Service) Upload(ctx context.Context, userID string, src io.Reader, size int64, contentType string) (domain.Upload, error) {
	if size > s.maxBytes {
		return domain.Upload{}, ErrFileTooLarge
	}
	ext, ok := allowedMIME[contentType]
	if !ok {
		return domain.Upload{}, ErrUnsupportedType
	}
	filename := uuid.NewString() + ext
	url, err := s.storage.Upload(ctx, filename, src)
	if err != nil {
		return domain.Upload{}, err
	}
	meta := domain.Upload{
		ID:               uuid.NewString(),
		UserID:           userID,
		Filename:         filename,
		URL:              url,
		ModerationStatus: domain.ModerationPending,
		CreatedAt:        time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, meta); err != nil {
		_ = s.storage.Delete(ctx, filename)
		return domain.Upload{}, err
	}
	s.moderator.ModerateAsync(meta)
	return meta, nil
}

// CleanupOrphans deletes uploads not linked to any report older than age.
func (s *Service) CleanupOrphans(ctx context.Context, age time.Duration) (int, error) {
	orphans, err := s.repo.FindOrphansOlderThan(ctx, age)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, u := range orphans {
		if err := s.storage.Delete(ctx, u.Filename); err != nil {
			slog.Warn("orphan cleanup: delete file failed", "filename", u.Filename, "error", err)
			continue
		}
		if err := s.repo.Delete(ctx, u.ID); err != nil {
			slog.Warn("orphan cleanup: delete record failed", "uploadId", u.ID, "error", err)
			continue
		}
		deleted++
	}
	return deleted, nil
}
