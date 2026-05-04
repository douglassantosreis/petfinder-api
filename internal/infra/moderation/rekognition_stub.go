//go:build !rekognition

package moderation

import (
	"context"
	"errors"

	moderationuc "github.com/yourname/go-backend/internal/usecase/moderation"
)

// RekognitionConfig mirrors the real config so server.go compiles without the SDK.
type RekognitionConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	S3Bucket        string
}

// RekognitionModerator is a placeholder that errors at startup when the
// rekognition build tag is not set.
// To enable: go get github.com/aws/aws-sdk-go-v2/service/rekognition && go build -tags rekognition
type RekognitionModerator struct{}

func NewRekognitionModerator(_ context.Context, _ RekognitionConfig) (*RekognitionModerator, error) {
	return nil, errors.New("rekognition not enabled: rebuild with -tags rekognition after running: go get github.com/aws/aws-sdk-go-v2/service/rekognition")
}

func (m *RekognitionModerator) Moderate(_ context.Context, _ string) (moderationuc.Result, error) {
	return moderationuc.Result{}, errors.New("rekognition not enabled")
}
