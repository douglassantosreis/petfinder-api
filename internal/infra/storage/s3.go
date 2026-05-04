package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string // CDN or direct S3 base URL
}

type S3Config struct {
	Bucket          string
	Region          string
	Endpoint        string // leave empty for AWS; set for MinIO / LocalStack
	AccessKeyID     string
	SecretAccessKey string
	PublicURL       string // e.g. https://cdn.example.com or https://bucket.s3.region.amazonaws.com
}

func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	clientOpts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		// MinIO, LocalStack, or any S3-compatible endpoint
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // required for MinIO
		})
	}

	client := s3.NewFromConfig(awsCfg, clientOpts...)

	publicURL := cfg.PublicURL
	if publicURL == "" {
		publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
	}

	return &S3Storage{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: publicURL,
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, filename string, src io.Reader) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
		Body:   src,
	})
	if err != nil {
		return "", fmt.Errorf("s3 put object: %w", err)
	}
	return s.publicURL + "/" + filename, nil
}

func (s *S3Storage) Delete(ctx context.Context, filename string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	})
	return err
}
