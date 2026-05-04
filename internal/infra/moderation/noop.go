package moderation

import (
	"context"

	moderationuc "github.com/yourname/go-backend/internal/usecase/moderation"
)

// NoOpModerator auto-approves every image.
// Used with local storage so development works without AWS credentials.
type NoOpModerator struct{}

func (m *NoOpModerator) Moderate(_ context.Context, _ string) (moderationuc.Result, error) {
	return moderationuc.Result{Approved: true}, nil
}
