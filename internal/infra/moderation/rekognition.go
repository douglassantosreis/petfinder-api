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

const minConfidence float32 = 70.0

// AWS Rekognition label taxonomy uses both names depending on model version.
var animalLabels = map[string]bool{
	"Animal":          true,
	"Animals and Pets": true,
}

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
	slog.Info("rekognition: moderating", "bucket", m.bucket, "filename", filename)
	s3obj := &types.S3Object{Bucket: aws.String(m.bucket), Name: aws.String(filename)}
	img := &types.Image{S3Object: s3obj}

	modOut, err := m.client.DetectModerationLabels(ctx, &rekognition.DetectModerationLabelsInput{
		Image:         img,
		MinConfidence: aws.Float32(minConfidence),
	})
	if err != nil {
		slog.Error("rekognition DetectModerationLabels failed, rejecting for safety", "filename", filename, "error", err)
		return moderationuc.Result{Approved: false, Reason: "moderation check unavailable"}, nil
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
		slog.Error("rekognition DetectLabels failed, rejecting for safety", "filename", filename, "error", err)
		return moderationuc.Result{Approved: false, Reason: "label detection unavailable"}, nil
	}

	slog.Info("rekognition: labels detected", "filename", filename, "count", len(labelsOut.Labels))
	if !hasAnimal(labelsOut.Labels) {
		slog.Info("image rejected: no animal detected", "filename", filename)
		return moderationuc.Result{Approved: false, Reason: "no animal detected in image"}, nil
	}

	slog.Info("rekognition: image approved", "filename", filename)
	return moderationuc.Result{Approved: true}, nil
}

func hasAnimal(labels []types.Label) bool {
	for _, label := range labels {
		name := aws.ToString(label.Name)
		if animalLabels[name] {
			slog.Debug("rekognition: animal label matched", "label", name)
			return true
		}
		for _, parent := range label.Parents {
			p := aws.ToString(parent.Name)
			if animalLabels[p] {
				slog.Debug("rekognition: animal label matched via parent", "label", name, "parent", p)
				return true
			}
		}
	}
	return false
}
