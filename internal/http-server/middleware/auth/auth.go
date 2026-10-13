package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"url-shortener/internal/lib/logger/sl"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned when a token cannot be parsed or is not valid.
var ErrInvalidToken = errors.New("invalid token")

// extractBearerToken extracts the auth token from the Authorization header.
func extractBearerToken(r *http.Request) string {
	const prefix = "Bearer "

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
}

// parseToken parses and validates a JWT issued by the SSO service.
func parseToken(tokenStr, appSecret string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(appSecret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// New creates an auth middleware. Requests without a valid token are rejected with 401.
func New(log *slog.Logger, appSecret string) func(next http.Handler) http.Handler {
	const op = "middleware.auth.New"

	log = log.With(slog.String("op", op))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearerToken(r)
			if tokenStr == "" {
				unauthorized(w, log, "token is missing")
				return
			}

			claims, err := parseToken(tokenStr, appSecret)
			if err != nil {
				log.Warn("failed to parse token", sl.Err(err))
				unauthorized(w, log, "invalid token")
				return
			}

			log.Info("user authorized", slog.Int64("uid", claims.UID))

			ctx := context.WithValue(r.Context(), uidKey, claims.UID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin allows the request only if the user is an admin according to SSO.
func RequireAdmin(log *slog.Logger, permProvider PermissionProvider) func(next http.Handler) http.Handler {
	const op = "middleware.auth.RequireAdmin"

	log = log.With(slog.String("op", op))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, ok := UIDFromContext(r.Context())
			if !ok {
				unauthorized(w, log, "user id is missing in context")
				return
			}

			isAdmin, err := permProvider.IsAdmin(r.Context(), uid)
			if err != nil {
				log.Error("failed to check if user is admin", sl.Err(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			if !isAdmin {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), isAdminKey, true)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter, log *slog.Logger, reason string) {
	log.Warn("unauthorized request", slog.String("reason", reason))
	w.Header().Set("WWW-Authenticate", `Bearer realm="url-shortener"`)
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
