package jwt

import (
	"context"
	"testing"
	"time"
)

func TestRefreshRotationRevokesPreviousToken(t *testing.T) {
	p := NewProvider("test-secret", NewInMemoryRevocationStore())
	ctx := context.Background()

	_, first, err := p.GenerateRefreshToken("user-1", time.Hour)
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	_, _, err = p.RotateRefreshToken(ctx, first, time.Hour)
	if err != nil {
		t.Fatalf("rotate refresh token: %v", err)
	}

	if _, _, _, err := p.ParseRefreshToken(ctx, first); err == nil {
		t.Fatal("expected first token to be revoked")
	}
}
