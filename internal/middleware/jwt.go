package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/0xFEE1DEADatm/goAuthAPI/internal/token"
)

type contextKey string

const userGUIDKey = contextKey("user_guid")

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "missing or invalid auth header", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		userGUID, err := token.ValidateAccessToken(tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userGUIDKey, userGUID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserGUIDFromContext(ctx context.Context) (string, error) {
	userGUID, ok := ctx.Value(userGUIDKey).(string)
	if !ok {
		return "", errors.New("user GUID not found in context")
	}
	return userGUID, nil
}
