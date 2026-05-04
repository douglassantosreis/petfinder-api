package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                    string
	MongoURI                string
	MongoDatabase           string
	JWTSecret               string
	AccessTokenTTLMinutes   int
	RefreshTokenTTLHours    int
	GoogleClientID          string
	GoogleClientSecret      string
	GoogleRedirectURL       string
	OAuthStateSigningSecret string

	// Storage — set STORAGE_PROVIDER=s3 to use S3, otherwise local filesystem
	StorageProvider string
	UploadDir       string // local only
	UploadBaseURL   string // public base URL (local or CDN)
	MaxUploadMB     int

	// S3 / S3-compatible (MinIO, LocalStack)
	S3Bucket           string
	S3Region           string
	S3Endpoint         string // leave empty for AWS; set for MinIO
	AWSAccessKeyID     string
	AWSSecretAccessKey string

}

func Load() Config {
	return Config{
		Port:                    getEnv("PORT", "8080"),
		MongoURI:                getEnv("MONGO_URI", "mongodb://admin:admin@localhost:27017/?authSource=admin"),
		MongoDatabase:           getEnv("MONGO_DATABASE", "petfinder"),
		JWTSecret:               getEnv("JWT_SECRET", "dev-secret"),
		AccessTokenTTLMinutes:   15,
		RefreshTokenTTLHours:    24 * 30,
		GoogleClientID:          os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:      os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:       os.Getenv("GOOGLE_REDIRECT_URL"),
		OAuthStateSigningSecret: getEnv("OAUTH_STATE_SECRET", "oauth-state-dev-secret"),
		StorageProvider:        getEnv("STORAGE_PROVIDER", "local"),
		UploadDir:              getEnv("UPLOAD_DIR", "./uploads"),
		UploadBaseURL:          getEnv("UPLOAD_BASE_URL", "http://localhost:8080"),
		MaxUploadMB:            getEnvInt("MAX_UPLOAD_MB", 5),
		S3Bucket:               os.Getenv("S3_BUCKET"),
		S3Region:               getEnv("S3_REGION", "us-east-1"),
		S3Endpoint:             os.Getenv("S3_ENDPOINT"),
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}
}

func getEnv(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(name string, fallback int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
