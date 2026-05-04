package jwt

import (
	"context"
	"errors"
	"time"

	gjwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type RevocationStore interface {
	Revoke(ctx context.Context, jti string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

type Provider struct {
	secret     []byte
	revocation RevocationStore
}

type claims struct {
	UserID    string `json:"uid"`
	TokenType string `json:"typ"`
	gjwt.RegisteredClaims
}

func NewProvider(secret string, revocation RevocationStore) *Provider {
	return &Provider{
		secret:     []byte(secret),
		revocation: revocation,
	}
}

func (p *Provider) GenerateAccessToken(userID string, expiresIn time.Duration) (string, error) {
	return p.signToken(userID, "access", expiresIn)
}

func (p *Provider) GenerateRefreshToken(userID string, expiresIn time.Duration) (string, string, error) {
	jti := uuid.NewString()
	token, err := p.signWithID(userID, "refresh", jti, expiresIn)
	return jti, token, err
}

func (p *Provider) ParseAccessToken(token string) (string, error) {
	c, err := p.parse(token)
	if err != nil {
		return "", err
	}
	if c.TokenType != "access" {
		return "", errors.New("invalid token type")
	}
	return c.UserID, nil
}

// ParseRefreshToken validates the token, checks revocation, and returns userID, jti, expiresAt.
func (p *Provider) ParseRefreshToken(ctx context.Context, token string) (string, string, time.Time, error) {
	c, err := p.parse(token)
	if err != nil {
		return "", "", time.Time{}, err
	}
	if c.TokenType != "refresh" {
		return "", "", time.Time{}, errors.New("invalid token type")
	}
	if c.ID == "" {
		return "", "", time.Time{}, errors.New("missing token id")
	}
	revoked, err := p.revocation.IsRevoked(ctx, c.ID)
	if err != nil {
		return "", "", time.Time{}, err
	}
	if revoked {
		return "", "", time.Time{}, errors.New("refresh token revoked")
	}
	var expiresAt time.Time
	if c.ExpiresAt != nil {
		expiresAt = c.ExpiresAt.Time
	}
	return c.UserID, c.ID, expiresAt, nil
}

func (p *Provider) RotateRefreshToken(ctx context.Context, currentToken string, expiresIn time.Duration) (string, string, error) {
	userID, currentJTI, expiresAt, err := p.ParseRefreshToken(ctx, currentToken)
	if err != nil {
		return "", "", err
	}
	if err := p.RevokeRefreshToken(ctx, currentJTI, expiresAt); err != nil {
		return "", "", err
	}
	return p.GenerateRefreshToken(userID, expiresIn)
}

func (p *Provider) RevokeRefreshToken(ctx context.Context, tokenID string, expiresAt time.Time) error {
	return p.revocation.Revoke(ctx, tokenID, expiresAt)
}

func (p *Provider) signToken(userID string, tokenType string, expiresIn time.Duration) (string, error) {
	return p.signWithID(userID, tokenType, "", expiresIn)
}

func (p *Provider) signWithID(userID string, tokenType string, jti string, expiresIn time.Duration) (string, error) {
	now := time.Now().UTC()
	c := claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: gjwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: gjwt.NewNumericDate(now.Add(expiresIn)),
			IssuedAt:  gjwt.NewNumericDate(now),
			ID:        jti,
		},
	}
	token := gjwt.NewWithClaims(gjwt.SigningMethodHS256, c)
	return token.SignedString(p.secret)
}

func (p *Provider) parse(tokenString string) (claims, error) {
	token, err := gjwt.ParseWithClaims(tokenString, &claims{}, func(token *gjwt.Token) (interface{}, error) {
		return p.secret, nil
	})
	if err != nil {
		return claims{}, err
	}
	if !token.Valid {
		return claims{}, errors.New("invalid token")
	}
	parsed, ok := token.Claims.(*claims)
	if !ok {
		return claims{}, errors.New("invalid claims")
	}
	return *parsed, nil
}
