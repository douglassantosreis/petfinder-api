package moderation

import (
	"context"
	"log/slog"

	uploadDomain "github.com/yourname/go-backend/internal/domain/upload"
)

// Moderator checks whether an image is acceptable.
type Moderator interface {
	Moderate(ctx context.Context, filename string) (Result, error)
}

// Result is the moderation outcome returned by a Moderator.
type Result struct {
	Approved        bool
	Reason          string
	ExplicitContent bool // true when inappropriate content was detected → triggers user ban
}

type UploadRepository interface {
	UpdateModerationStatus(ctx context.Context, uploadID string, status uploadDomain.ModerationStatus, reason string) error
	FindByReport(ctx context.Context, reportID string) ([]uploadDomain.Upload, error)
}

type ReportRepository interface {
	SetVisible(ctx context.Context, reportID string, visible bool) error
}

type UserBanner interface {
	Ban(ctx context.Context, userID string) error
}

type Service struct {
	moderator Moderator
	uploads   UploadRepository
	reports   ReportRepository
	users     UserBanner
}

func NewService(moderator Moderator, uploads UploadRepository, reports ReportRepository, users UserBanner) *Service {
	return &Service{moderator: moderator, uploads: uploads, reports: reports, users: users}
}

// ModerateAsync runs moderation in a goroutine — the upload response is not blocked.
func (s *Service) ModerateAsync(upload uploadDomain.Upload) {
	go func() {
		if err := s.moderate(context.Background(), upload); err != nil {
			slog.Error("moderation failed", "uploadId", upload.ID, "error", err)
		}
	}()
}

func (s *Service) moderate(ctx context.Context, upload uploadDomain.Upload) error {
	result, err := s.moderator.Moderate(ctx, upload.Filename)
	if err != nil {
		return err
	}

	status := uploadDomain.ModerationApproved
	reason := ""
	if !result.Approved {
		status = uploadDomain.ModerationRejected
		reason = result.Reason
	}

	if err := s.uploads.UpdateModerationStatus(ctx, upload.ID, status, reason); err != nil {
		return err
	}

	slog.Info("moderation result", "uploadId", upload.ID, "status", status, "reason", reason)

	if result.ExplicitContent {
		slog.Warn("explicit content detected, banning user", "userID", upload.UserID, "uploadId", upload.ID)
		if err := s.users.Ban(ctx, upload.UserID); err != nil {
			slog.Error("failed to ban user", "userID", upload.UserID, "error", err)
		}
	}

	// When approved and linked to a report, check if all photos are now approved.
	if status == uploadDomain.ModerationApproved && upload.ReportID != "" {
		s.maybeSetReportVisible(ctx, upload.ReportID)
	}

	return nil
}

func (s *Service) maybeSetReportVisible(ctx context.Context, reportID string) {
	uploads, err := s.uploads.FindByReport(ctx, reportID)
	if err != nil {
		slog.Warn("moderation: could not fetch uploads for report", "reportId", reportID, "error", err)
		return
	}
	for _, u := range uploads {
		if u.ModerationStatus != uploadDomain.ModerationApproved {
			return
		}
	}
	if err := s.reports.SetVisible(ctx, reportID, true); err != nil {
		slog.Warn("moderation: could not set report visible", "reportId", reportID, "error", err)
	} else {
		slog.Info("report is now visible", "reportId", reportID)
	}
}
