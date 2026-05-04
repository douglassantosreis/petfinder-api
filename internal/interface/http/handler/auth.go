package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/yourname/go-backend/internal/domain/user"
	authuc "github.com/yourname/go-backend/internal/usecase/auth"
)

type AuthHandler struct {
	service *authuc.Service
}

func NewAuthHandler(service *authuc.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

// Register godoc
// @Summary Register with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body RegisterRequest true "Registration payload"
// @Success 201 {object} OAuthCallbackResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /v1/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeJSON(r, &req); err != nil || req.Name == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "name, email and password are required", http.StatusBadRequest)
		return
	}
	slog.Info("register attempt", "name", req.Name, "email", req.Email)
	u, accessToken, refreshToken, err := h.service.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrEmailTaken) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusCreated, OAuthCallbackResponse{
		User:         toUserResponse(u),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Login godoc
// @Summary Login with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body LoginRequest true "Login payload"
// @Success 200 {object} OAuthCallbackResponse
// @Failure 400 {object} ErrorResponse "missing fields"
// @Failure 401 {object} ErrorResponse "invalid email or password"
// @Failure 423 {object} ErrorResponse "account suspended for policy violation"
// @Router /v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeJSON(r, &req); err != nil || req.Email == "" || req.Password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}
	u, accessToken, refreshToken, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrUserBanned) {
			http.Error(w, err.Error(), http.StatusLocked)
			return
		}
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}
	respondJSON(w, http.StatusOK, OAuthCallbackResponse{
		User:         toUserResponse(u),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// StartGoogleOAuth godoc
// @Summary Start Google OAuth flow
// @Tags auth
// @Produce json
// @Success 200 {object} StartOAuthResponse
// @Router /v1/auth/oauth/google/start [get]
func (h *AuthHandler) StartGoogleOAuth(w http.ResponseWriter, _ *http.Request) {
	authURL, state := h.service.StartOAuth()
	slog.Info("oauth start issued", "state_len", len(state), "url_has_state", len(state) > 0)
	respondJSON(w, http.StatusOK, StartOAuthResponse{AuthURL: authURL, State: state})
}

// GoogleCallback godoc
// @Summary Handle Google OAuth callback
// @Tags auth
// @Produce json
// @Param code query string true "Authorization code"
// @Param state query string true "OAuth state for CSRF protection"
// @Success 200 {object} OAuthCallbackResponse
// @Failure 400 {object} ErrorResponse "missing code or OAuth error"
// @Failure 401 {object} ErrorResponse "invalid state or token exchange failed"
// @Failure 423 {object} ErrorResponse "account suspended for policy violation"
// @Router /v1/auth/oauth/google/callback [get]
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
		desc := r.URL.Query().Get("error_description")
		slog.Warn("oauth callback error from google", "error", oauthErr, "description", desc)
		http.Error(w, "oauth error: "+oauthErr, http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		slog.Warn("oauth callback missing code", "raw_query", r.URL.RawQuery)
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}
	state := r.URL.Query().Get("state")
	if state == "" {
		slog.Warn("oauth callback: state absent, CSRF protection skipped", "raw_query", r.URL.RawQuery)
	}
	u, accessToken, refreshToken, err := h.service.OAuthCallback(r.Context(), code, state)
	if err != nil {
		if errors.Is(err, user.ErrUserBanned) {
			http.Error(w, err.Error(), http.StatusLocked)
			return
		}
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	respondJSON(w, http.StatusOK, OAuthCallbackResponse{
		User:         toUserResponse(u),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Refresh godoc
// @Summary Refresh session tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body RefreshRequest true "Refresh token"
// @Success 200 {object} RefreshResponse
// @Failure 400 {object} ErrorResponse "missing refresh token"
// @Failure 401 {object} ErrorResponse "invalid or expired refresh token"
// @Failure 423 {object} ErrorResponse "account suspended for policy violation"
// @Router /v1/auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := decodeJSON(r, &req); err != nil || req.RefreshToken == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	accessToken, refreshToken, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, user.ErrUserBanned) {
			http.Error(w, err.Error(), http.StatusLocked)
			return
		}
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	respondJSON(w, http.StatusOK, RefreshResponse{AccessToken: accessToken, RefreshToken: refreshToken})
}

// Logout godoc
// @Summary Logout user session
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Param payload body RefreshRequest true "Refresh token"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /v1/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := decodeJSON(r, &req); err != nil || req.RefreshToken == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := h.service.Logout(r.Context(), req.RefreshToken); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toUserResponse(u user.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		City:      u.City,
		State:     u.State,
	}
}
