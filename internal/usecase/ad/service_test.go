package ad

import (
	"context"
	"testing"

	domain "github.com/yourname/go-backend/internal/domain/ad"
)

type fakeRepo struct {
	item domain.FoundAnimalReport
}

func (f *fakeRepo) Create(_ context.Context, report domain.FoundAnimalReport) (domain.FoundAnimalReport, error) {
	f.item = report
	return report, nil
}
func (f *fakeRepo) GetByID(_ context.Context, _ string) (domain.FoundAnimalReport, error) {
	return f.item, nil
}
func (f *fakeRepo) ListOpen(_ context.Context, _, _ int, _ *GeoFilter) ([]domain.FoundAnimalReport, error) {
	return []domain.FoundAnimalReport{f.item}, nil
}
func (f *fakeRepo) Update(_ context.Context, report domain.FoundAnimalReport) (domain.FoundAnimalReport, error) {
	f.item = report
	return report, nil
}
func (f *fakeRepo) SetVisible(_ context.Context, _ string, visible bool) error {
	f.item.Visible = visible
	return nil
}

type fakeUploadRepo struct{}

func (f *fakeUploadRepo) ValidateOwnership(_ context.Context, _ []string, _, _ string) error {
	return nil
}
func (f *fakeUploadRepo) AssignToReport(_ context.Context, _ []string, _ string) error {
	return nil
}
func (f *fakeUploadRepo) AllApproved(_ context.Context, _ []string) (bool, error) {
	return true, nil
}

func svc() *Service {
	return NewService(&fakeRepo{}, &fakeUploadRepo{})
}

func TestUpdateFailsForNonOwner(t *testing.T) {
	repo := &fakeRepo{item: domain.FoundAnimalReport{ID: "report-1", OwnerID: "owner-1"}}
	s := NewService(repo, &fakeUploadRepo{})

	_, err := s.Update(context.Background(), "owner-2", "report-1", domain.FoundAnimalReport{Title: "new"})
	if err == nil {
		t.Fatal("expected forbidden error")
	}
}

func TestCreateRejectsTooManyPhotos(t *testing.T) {
	s := svc()
	_, err := s.Create(context.Background(), CreateInput{
		OwnerID: "u1",
		Photos:  []string{"a.jpg", "b.jpg", "c.jpg", "d.jpg"},
	})
	if err != ErrTooManyPhotos {
		t.Fatalf("expected ErrTooManyPhotos, got %v", err)
	}
}
