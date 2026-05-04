package ad

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	domain "github.com/yourname/go-backend/internal/domain/ad"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrTooManyPhotos = errors.New("maximum of 3 photos per report")
)

const maxPhotos = 3

// GeoFilter restricts and sorts results by proximity.
// RadiusKm defaults to 50 when zero.
type GeoFilter struct {
	Latitude  float64
	Longitude float64
	RadiusKm  float64
}

func (g GeoFilter) RadiusMeters() float64 {
	if g.RadiusKm <= 0 {
		return 500
	}
	return g.RadiusKm * 1000
}

type Repository interface {
	Create(ctx context.Context, report domain.FoundAnimalReport) (domain.FoundAnimalReport, error)
	GetByID(ctx context.Context, id string) (domain.FoundAnimalReport, error)
	ListOpen(ctx context.Context, page, pageSize int, geo *GeoFilter) ([]domain.FoundAnimalReport, error)
	Update(ctx context.Context, report domain.FoundAnimalReport) (domain.FoundAnimalReport, error)
	SetVisible(ctx context.Context, reportID string, visible bool) error
}

// UploadRepository is the subset of upload operations needed by this service.
type UploadRepository interface {
	ValidateOwnership(ctx context.Context, urls []string, userID, currentReportID string) error
	AssignToReport(ctx context.Context, urls []string, reportID string) error
	// AllApproved returns true when every URL has moderationStatus "approved".
	AllApproved(ctx context.Context, urls []string) (bool, error)
}

type Service struct {
	repo    Repository
	uploads UploadRepository
}

func NewService(repo Repository, uploads UploadRepository) *Service {
	return &Service{repo: repo, uploads: uploads}
}

type CreateInput struct {
	OwnerID               string
	PetType               string
	Title                 string
	Description           string
	Characteristics       []string
	Latitude              float64
	Longitude             float64
	Photos                []string
	IsShelteredByReporter bool
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.FoundAnimalReport, error) {
	if len(input.Photos) > maxPhotos {
		return domain.FoundAnimalReport{}, ErrTooManyPhotos
	}
	if len(input.Photos) > 0 {
		if err := s.uploads.ValidateOwnership(ctx, input.Photos, input.OwnerID, ""); err != nil {
			return domain.FoundAnimalReport{}, err
		}
	}
	now := time.Now().UTC()
	// Reports with photos start hidden until moderation approves all images.
	visible := len(input.Photos) == 0
	report, err := s.repo.Create(ctx, domain.FoundAnimalReport{
		ID:                    uuid.NewString(),
		OwnerID:               input.OwnerID,
		PetType:               input.PetType,
		Title:                 input.Title,
		Description:           input.Description,
		Characteristics:       input.Characteristics,
		LastSeenLocation:      domain.NewGeoPoint(input.Latitude, input.Longitude),
		Photos:                input.Photos,
		IsShelteredByReporter: input.IsShelteredByReporter,
		Status:                domain.StatusOpen,
		Visible:               visible,
		CreatedAt:             now,
		UpdatedAt:             now,
	})
	if err != nil {
		return domain.FoundAnimalReport{}, err
	}
	if len(input.Photos) > 0 {
		if err := s.uploads.AssignToReport(ctx, input.Photos, report.ID); err != nil {
			slog.Warn("assign photos to report failed", "reportId", report.ID, "error", err)
		}
		// Photos may already be approved (e.g. NoOp moderator ran before report creation).
		// Set visible immediately in that case; otherwise moderation will do it later.
		if ok, err := s.uploads.AllApproved(ctx, input.Photos); err == nil && ok {
			if err := s.repo.SetVisible(ctx, report.ID, true); err != nil {
				slog.Warn("set report visible failed", "reportId", report.ID, "error", err)
			} else {
				report.Visible = true
			}
		}
	}
	return report, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (domain.FoundAnimalReport, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListOpen(ctx context.Context, page, pageSize int, geo *GeoFilter) ([]domain.FoundAnimalReport, error) {
	page, pageSize = normalizePage(page, pageSize)
	return s.repo.ListOpen(ctx, page, pageSize, geo)
}

func (s *Service) Update(ctx context.Context, userID string, reportID string, patch domain.FoundAnimalReport) (domain.FoundAnimalReport, error) {
	current, err := s.repo.GetByID(ctx, reportID)
	if err != nil {
		return domain.FoundAnimalReport{}, err
	}
	if current.OwnerID != userID {
		return domain.FoundAnimalReport{}, ErrForbidden
	}
	if len(patch.Photos) > maxPhotos {
		return domain.FoundAnimalReport{}, ErrTooManyPhotos
	}
	if len(patch.Photos) > 0 {
		// Allow photos already on this report + new unassigned ones from this user.
		if err := s.uploads.ValidateOwnership(ctx, patch.Photos, userID, reportID); err != nil {
			return domain.FoundAnimalReport{}, err
		}
	}
	current.Title = patch.Title
	current.Description = patch.Description
	current.Characteristics = patch.Characteristics
	current.LastSeenLocation = patch.LastSeenLocation
	current.Photos = patch.Photos
	current.IsShelteredByReporter = patch.IsShelteredByReporter
	current.UpdatedAt = time.Now().UTC()
	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return domain.FoundAnimalReport{}, err
	}
	if len(patch.Photos) > 0 {
		if err := s.uploads.AssignToReport(ctx, patch.Photos, reportID); err != nil {
			slog.Warn("assign photos to report failed", "reportId", reportID, "error", err)
		}
	}
	return updated, nil
}

func (s *Service) Resolve(ctx context.Context, userID, reportID string) (domain.FoundAnimalReport, error) {
	return s.setStatus(ctx, userID, reportID, domain.StatusResolved)
}

func (s *Service) Archive(ctx context.Context, userID, reportID string) (domain.FoundAnimalReport, error) {
	return s.setStatus(ctx, userID, reportID, domain.StatusArchived)
}

func (s *Service) setStatus(ctx context.Context, userID, reportID string, status domain.Status) (domain.FoundAnimalReport, error) {
	current, err := s.repo.GetByID(ctx, reportID)
	if err != nil {
		return domain.FoundAnimalReport{}, err
	}
	if current.OwnerID != userID {
		return domain.FoundAnimalReport{}, ErrForbidden
	}
	current.Status = status
	current.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, current)
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
