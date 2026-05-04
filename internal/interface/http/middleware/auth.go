package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

type contextKey string

const userIDKey contextKey = "user_id"

type AccessTokenParser interface {
	ParseAccessToken(token string) (string, error)
}

type UserBanChecker interface {
	IsBanned(ctx context.Context, userID string) (bool, error)
}

func AuthRequired(parser AccessTokenParser, banChecker UserBanChecker, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := parser.ParseAccessToken(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		banned, err := banChecker.IsBanned(r.Context(), userID)
		if err != nil {
			slog.Warn("auth: ban check failed", "userID", userID, "error", err)
		} else if banned {
			http.Error(w, "account suspended for policy violation", http.StatusLocked)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(userIDKey).(string); ok {
		return value
	}
	return ""
}
