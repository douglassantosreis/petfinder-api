package user

import (
	"context"
	"errors"
	"time"

	domain "github.com/yourname/go-backend/internal/domain/user"
)

var ErrNotFound = errors.New("user not found")

type Repository interface {
	GetByID(ctx context.Context, id string) (domain.User, error)
	UpdateMe(ctx context.Context, id string, name string, city string, state string) (domain.User, error)
	SoftDelete(ctx context.Context, id string, at time.Time) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Me(ctx context.Context, userID string) (domain.User, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *Service) UpdateMe(ctx context.Context, userID string, name string, city string, state string) (domain.User, error) {
	return s.repo.UpdateMe(ctx, userID, name, city, state)
}

func (s *Service) DeleteMe(ctx context.Context, userID string) error {
	return s.repo.SoftDelete(ctx, userID, time.Now().UTC())
}
