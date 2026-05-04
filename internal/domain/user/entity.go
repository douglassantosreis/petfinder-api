package user

import (
	"errors"
	"time"
)

var (
	ErrEmailTaken       = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNotFound         = errors.New("user not found")
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusBanned   Status = "banned"
)

var ErrUserBanned = errors.New("account suspended for policy violation")

type User struct {
	ID           string    `bson:"_id"          json:"id"`
	Name         string    `bson:"name"         json:"name"`
	Email        string    `bson:"email"        json:"email"`
	AvatarURL    string    `bson:"avatarUrl"    json:"avatarUrl,omitempty"`
	City         string    `bson:"city"         json:"city,omitempty"`
	State        string    `bson:"state"        json:"state,omitempty"`
	OAuthProvider string   `bson:"oauthProvider" json:"oauthProvider,omitempty"`
	OAuthSubject  string   `bson:"oauthSubject"  json:"oauthSubject,omitempty"`
	PasswordHash string    `bson:"passwordHash" json:"-"` // never serialized
	Status       Status    `bson:"status"       json:"status"`
	CreatedAt    time.Time `bson:"createdAt"    json:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt"    json:"updatedAt"`
}
