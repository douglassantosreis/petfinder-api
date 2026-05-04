package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yourname/go-backend/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

type OAuthProfile struct {
	Provider string
	Subject  string
	Email    string
	Name     string
	Avatar   string
}

type OAuthProvider interface {
	StartURL(state string) string
	ExchangeCode(ctx context.Context, code string) (OAuthProfile, error)
}

type StateValidator interface {
	Generate() string
	Validate(state string) error
}

type UserRepository interface {
	UpsertByOAuth(ctx context.Context, input user.User) (user.User, error)
	CreateWithPassword(ctx context.Context, u user.User) (user.User, error)
	GetByID(ctx context.Context, id string) (user.User, error)
	GetByEmail(ctx context.Context, email string) (user.User, error)
}

type TokenManager interface {
	GenerateAccessToken(userID string, expiresIn time.Duration) (string, error)
	GenerateRefreshToken(userID string, expiresIn time.Duration) (string, string, error)
	ParseAccessToken(token string) (string, error)
	ParseRefreshToken(ctx context.Context, token string) (userID string, jti string, expiresAt time.Time, err error)
	RotateRefreshToken(ctx context.Context, currentToken string, expiresIn time.Duration) (string, string, error)
	RevokeRefreshToken(ctx context.Context, tokenID string, expiresAt time.Time) error
}

type Service struct {
	users      UserRepository
	provider   OAuthProvider
	stateMgr   StateValidator
	tokenMgr   TokenManager
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(users UserRepository, provider OAuthProvider, stateMgr StateValidator, tokenMgr TokenManager, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		users:      users,
		provider:   provider,
		stateMgr:   stateMgr,
		tokenMgr:   tokenMgr,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Register creates a new email/password account and returns tokens.
func (s *Service) Register(ctx context.Context, name, email, password string) (user.User, string, string, error) {
	if err := validateEmail(email); err != nil {
		return user.User{}, "", "", err
	}
	if len(password) < 8 {
		return user.User{}, "", "", errors.New("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return user.User{}, "", "", err
	}
	now := time.Now().UTC()
	u, err := s.users.CreateWithPassword(ctx, user.User{
		ID:           uuid.NewString(),
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Status:       user.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return user.User{}, "", "", err
	}
	return s.issueTokens(u)
}

// Login authenticates an email/password user and returns tokens.
func (s *Service) Login(ctx context.Context, email, password string) (user.User, string, string, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return user.User{}, "", "", user.ErrInvalidCredentials
	}
	if u.PasswordHash == "" {
		// OAuth-only account — cannot log in with password
		return user.User{}, "", "", user.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return user.User{}, "", "", user.ErrInvalidCredentials
	}
	return s.issueTokens(u)
}

func (s *Service) StartOAuth() (authURL string, state string) {
	state = s.stateMgr.Generate()
	return s.provider.StartURL(state), state
}

func (s *Service) OAuthCallback(ctx context.Context, code, state string) (user.User, string, string, error) {
	if state != "" {
		if err := s.stateMgr.Validate(state); err != nil {
			return user.User{}, "", "", errors.New("invalid oauth state")
		}
	}
	profile, err := s.provider.ExchangeCode(ctx, code)
	if err != nil {
		return user.User{}, "", "", err
	}
	now := time.Now().UTC()
	u, err := s.users.UpsertByOAuth(ctx, user.User{
		ID:            uuid.NewString(),
		Name:          profile.Name,
		Email:         profile.Email,
		AvatarURL:     profile.Avatar,
		OAuthProvider: profile.Provider,
		OAuthSubject:  profile.Subject,
		Status:        user.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return user.User{}, "", "", err
	}
	return s.issueTokens(u)
}

func (s *Service) Refresh(ctx context.Context, currentRefreshToken string) (string, string, error) {
	userID, _, _, err := s.tokenMgr.ParseRefreshToken(ctx, currentRefreshToken)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}
	_, newToken, err := s.tokenMgr.RotateRefreshToken(ctx, currentRefreshToken, s.refreshTTL)
	if err != nil {
		return "", "", err
	}
	accessToken, err := s.tokenMgr.GenerateAccessToken(userID, s.accessTTL)
	if err != nil {
		return "", "", err
	}
	return accessToken, newToken, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	_, tokenID, expiresAt, err := s.tokenMgr.ParseRefreshToken(ctx, refreshToken)
	if err != nil {
		return err
	}
	return s.tokenMgr.RevokeRefreshToken(ctx, tokenID, expiresAt)
}

func (s *Service) issueTokens(u user.User) (user.User, string, string, error) {
	accessToken, err := s.tokenMgr.GenerateAccessToken(u.ID, s.accessTTL)
	if err != nil {
		return user.User{}, "", "", err
	}
	_, refreshToken, err := s.tokenMgr.GenerateRefreshToken(u.ID, s.refreshTTL)
	if err != nil {
		return user.User{}, "", "", err
	}
	return u, accessToken, refreshToken, nil
}

func validateEmail(email string) error {
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return errors.New("invalid email address")
	}
	return nil
}
