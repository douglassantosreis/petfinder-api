package oauth

import (
	"context"
	"encoding/json"
	"errors"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	authuc "github.com/yourname/go-backend/internal/usecase/auth"
)

type GoogleProvider struct {
	conf *oauth2.Config
}

func NewGoogleProvider(clientID, clientSecret, redirectURL string) *GoogleProvider {
	return &GoogleProvider{
		conf: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "profile", "email"},
		},
	}
}

func (g *GoogleProvider) StartURL(state string) string {
	return g.conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (g *GoogleProvider) ExchangeCode(ctx context.Context, code string) (authuc.OAuthProfile, error) {
	token, err := g.conf.Exchange(ctx, code)
	if err != nil {
		return authuc.OAuthProfile{}, err
	}
	client := g.conf.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return authuc.OAuthProfile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return authuc.OAuthProfile{}, errors.New("failed to fetch google user profile")
	}
	var payload struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return authuc.OAuthProfile{}, err
	}
	return authuc.OAuthProfile{
		Provider: "google",
		Subject:  payload.ID,
		Email:    payload.Email,
		Name:     payload.Name,
		Avatar:   payload.Picture,
	}, nil
}
