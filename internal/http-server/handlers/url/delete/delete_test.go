package delete_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/http-server/handlers/url/delete/mocks"
	"url-shortener/internal/http-server/middleware/auth"
	apiresp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/slogdiscard"
	"url-shortener/internal/storage"
)

func TestDeleteHandler(t *testing.T) {
	t.Parallel()

	const (
		alias = "abc"
		uid   = int64(1)
	)

	cases := []struct {
		name       string
		withUser   bool
		ownerID    int64
		urlErr     error
		callAdmin  bool
		isAdmin    bool
		isAdminErr error
		callDelete bool
		deleteErr  error
		wantCode   int
		respError  string
	}{
		{
			name:       "Owner deletes",
			withUser:   true,
			ownerID:    uid,
			callDelete: true,
			wantCode:   http.StatusOK,
		},
		{
			name:       "Admin deletes others",
			withUser:   true,
			ownerID:    uid + 1,
			callAdmin:  true,
			isAdmin:    true,
			callDelete: true,
			wantCode:   http.StatusOK,
		},
		{
			name:      "Not owner and not admin",
			withUser:  true,
			ownerID:   uid + 1,
			callAdmin: true,
			isAdmin:   false,
			wantCode:  http.StatusForbidden,
			respError: "forbidden",
		},
		{
			name:      "Not found",
			withUser:  true,
			urlErr:    storage.ErrURLNotFound,
			wantCode:  http.StatusOK,
			respError: "not found",
		},
		{
			name:       "Admin check error",
			withUser:   true,
			ownerID:    uid + 1,
			callAdmin:  true,
			isAdminErr: errors.New("sso down"),
			wantCode:   http.StatusOK,
			respError:  "internal error",
		},
		{
			name:      "Unauthorized",
			withUser:  false,
			wantCode:  http.StatusUnauthorized,
			respError: "user is not authorized",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlDeleterMock := mocks.NewURLDeleter(t)
			permProviderMock := mocks.NewPermissionProvider(t)

			if tc.withUser {
				urlDeleterMock.On("URL", alias).Return("https://google.com", tc.ownerID, tc.urlErr).Once()
			}
			if tc.callAdmin {
				permProviderMock.On("IsAdmin", mock.Anything, uid).Return(tc.isAdmin, tc.isAdminErr).Once()
			}
			if tc.callDelete {
				urlDeleterMock.On("DeleteURL", alias).Return(tc.deleteErr).Once()
			}

			handler := delete.New(slogdiscard.NewDiscardLogger(), urlDeleterMock, permProviderMock)

			req, err := http.NewRequest(http.MethodDelete, "/url/"+alias, nil)
			require.NoError(t, err)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("alias", alias)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			if tc.withUser {
				req = req.WithContext(auth.WithUserID(req.Context(), uid))
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.wantCode, rr.Code)

			var body apiresp.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))

			require.Equal(t, tc.respError, body.Error)

			wantStatus := "OK"
			if tc.respError != "" {
				wantStatus = "Error"
			}
			require.Equal(t, wantStatus, body.Status)
		})
	}
}
