package redirect_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/redirect/mocks"
	apiresp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/slogdiscard"
	"url-shortener/internal/storage"
)

func TestRedirectHandler(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		alias        string
		withRoute    bool
		url          string
		err          error
		wantCode     int
		wantLocation string
		respError    string
	}{
		{
			name:         "Success",
			alias:        "abc",
			withRoute:    true,
			url:          "https://google.com",
			wantCode:     http.StatusFound,
			wantLocation: "https://google.com",
		},
		{
			name:      "Alias is empty",
			withRoute: false,
			wantCode:  http.StatusOK,
			respError: "invalid request",
		},
		{
			name:      "Not found",
			alias:     "abc",
			withRoute: true,
			err:       storage.ErrURLNotFound,
			wantCode:  http.StatusOK,
			respError: "not found",
		},
		{
			name:      "Internal error",
			alias:     "abc",
			withRoute: true,
			err:       errors.New("boom"),
			wantCode:  http.StatusOK,
			respError: "internal error",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlGetterMock := mocks.NewURLGetter(t)
			if tc.withRoute {
				urlGetterMock.On("GetURL", tc.alias).Return(tc.url, tc.err).Once()
			}

			handler := redirect.New(slogdiscard.NewDiscardLogger(), urlGetterMock)

			req, err := http.NewRequest(http.MethodGet, "/"+tc.alias, nil)
			require.NoError(t, err)

			if tc.withRoute {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("alias", tc.alias)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.wantCode, rr.Code)

			if tc.wantLocation != "" {
				require.Equal(t, tc.wantLocation, rr.Header().Get("Location"))
			}

			if tc.respError != "" {
				var body apiresp.Response
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
				require.Equal(t, tc.respError, body.Error)
			}
		})
	}
}
