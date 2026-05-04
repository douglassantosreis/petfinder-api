package oauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type StateManager struct {
	secret []byte
}

func NewStateManager(secret string) *StateManager {
	return &StateManager{secret: []byte(secret)}
}

// Generate produces a self-verifiable state token: nonce.hmac(nonce).
// No server-side session needed — the HMAC proves we issued the nonce.
func (m *StateManager) Generate() string {
	nonce := uuid.NewString()
	return nonce + "." + m.sign(nonce)
}

func (m *StateManager) Validate(state string) error {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) != 2 {
		return errors.New("invalid oauth state format")
	}
	expected := m.sign(parts[0])
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return errors.New("invalid oauth state signature")
	}
	return nil
}

func (m *StateManager) sign(nonce string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(nonce))
	return hex.EncodeToString(mac.Sum(nil))
}
