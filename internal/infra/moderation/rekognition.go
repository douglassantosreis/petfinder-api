//go:build rekognition

package moderation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	moderationuc "github.com/yourname/go-backend/internal/usecase/moderation"
)

const (
	minConfidence     float32 = 70.0
	animalParentLabel         = "Animal"
)

type RekognitionConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	S3Bucket        string
}

type RekognitionModerator struct {
	client *rekognition.Client
	bucket string
}

func NewRekognitionModerator(ctx context.Context, cfg RekognitionConfig) (*RekognitionModerator, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config for rekognition: %w", err)
	}
	return &RekognitionModerator{
		client: rekognition.NewFromConfig(awsCfg),
		bucket: cfg.S3Bucket,
	}, nil
}

func (m *RekognitionModerator) Moderate(ctx context.Context, filename string) (moderationuc.Result, error) {
	s3obj := &types.S3Object{Bucket: aws.String(m.bucket), Name: aws.String(filename)}
	img := &types.Image{S3Object: s3obj}

	modOut, err := m.client.DetectModerationLabels(ctx, &rekognition.DetectModerationLabelsInput{
		Image:         img,
		MinConfidence: aws.Float32(minConfidence),
	})
	if err != nil {
		slog.Warn("rekognition DetectModerationLabels failed, auto-approving", "filename", filename, "error", err)
		return moderationuc.Result{Approved: true}, nil
	}
	if len(modOut.ModerationLabels) > 0 {
		reason := aws.ToString(modOut.ModerationLabels[0].Name)
		slog.Info("image rejected: inappropriate content", "filename", filename, "reason", reason)
		return moderationuc.Result{Approved: false, Reason: "inappropriate content: " + reason}, nil
	}

	labelsOut, err := m.client.DetectLabels(ctx, &rekognition.DetectLabelsInput{
		Image:         img,
		MinConfidence: aws.Float32(minConfidence),
	})
	if err != nil {
		slog.Warn("rekognition DetectLabels failed, auto-approving", "filename", filename, "error", err)
		return moderationuc.Result{Approved: true}, nil
	}
	if !hasAnimal(labelsOut.Labels) {
		slog.Info("image rejected: no animal detected", "filename", filename)
		return moderationuc.Result{Approved: false, Reason: "no animal detected in image"}, nil
	}

	return moderationuc.Result{Approved: true}, nil
}

func hasAnimal(labels []types.Label) bool {
	for _, label := range labels {
		if aws.ToString(label.Name) == animalParentLabel {
			return true
		}
		for _, parent := range label.Parents {
			if aws.ToString(parent.Name) == animalParentLabel {
				return true
			}
		}
	}
	return false
}
