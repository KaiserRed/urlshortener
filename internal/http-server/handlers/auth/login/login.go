package login

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	ssogrpc "url-shortener/internal/clients/sso/grpc"
	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"

	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type Response struct {
	resp.Response
	Token string `json:"token,omitempty"`
}

// TokenProvider issues auth tokens for the given credentials.
type TokenProvider interface {
	Login(ctx context.Context, email, password string, appID int) (string, error)
}

// New creates a handler that logs a user in and returns an auth token.
func New(log *slog.Logger, provider TokenProvider, appID int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.auth.login.New"

		log = log.With(slog.String("op", op))

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, resp.Error("failed to decode request"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			log.Error("invalid request", sl.Err(err))

			var validateErr validator.ValidationErrors
			if errors.As(err, &validateErr) {
				render.JSON(w, r, resp.ValidationError(validateErr))
			} else {
				render.JSON(w, r, resp.Error("invalid request"))
			}
			return
		}

		token, err := provider.Login(r.Context(), req.Email, req.Password, appID)
		if err != nil {
			if errors.Is(err, ssogrpc.ErrInvalidCredentials) {
				log.Info("invalid credentials")
				render.JSON(w, r, resp.Error("invalid credentials"))
				return
			}
			log.Error("failed to login", sl.Err(err))
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		render.JSON(w, r, Response{
			Response: resp.OK(),
			Token:    token,
		})
	}
}
