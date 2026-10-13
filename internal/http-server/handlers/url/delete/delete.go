package delete

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"url-shortener/internal/http-server/middleware/auth"
	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// URLDeleter provides URL ownership lookup and deletion.
type URLDeleter interface {
	URL(alias string) (url string, userID int64, err error)
	DeleteURL(alias string) error
}

// PermissionProvider resolves user permissions. In production it is the SSO client.
type PermissionProvider interface {
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

// New creates a handler that deletes a URL if the caller is its owner or an admin.
func New(log *slog.Logger, urlDeleter URLDeleter, permProvider PermissionProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.delete.New"

		log = log.With(slog.String("op", op))

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			render.JSON(w, r, resp.Error("invalid request"))
			return
		}

		uid, ok := auth.UIDFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Error("user is not authorized"))
			return
		}

		_, ownerID, err := urlDeleter.URL(alias)
		if err != nil {
			if errors.Is(err, storage.ErrURLNotFound) {
				render.JSON(w, r, resp.Error("not found"))
				return
			}

			log.Error("failed to get url", sl.Err(err))
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		if ownerID != uid {
			isAdmin, err := permProvider.IsAdmin(r.Context(), uid)
			if err != nil {
				log.Error("failed to check if user is admin", sl.Err(err))
				render.JSON(w, r, resp.Error("internal error"))
				return
			}
			if !isAdmin {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Error("forbidden"))
				return
			}
		}

		if err := urlDeleter.DeleteURL(alias); err != nil {
			if errors.Is(err, storage.ErrURLNotFound) {
				render.JSON(w, r, resp.Error("not found"))
				return
			}

			log.Error("failed to delete url", sl.Err(err))
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		render.JSON(w, r, resp.OK())
	}
}
