package save_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/save/mocks"
	"url-shortener/internal/http-server/middleware/auth"
	"url-shortener/internal/lib/logger/slogdiscard"
)

func TestSaveHandler(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		url       string
		alias     string
		withUser  bool
		callSave  bool
		respError string
		mockError error
		wantCode  int
		wantAlias string
	}{
		{
			name:      "Success with alias",
			url:       "https://google.com",
			alias:     "test_alias",
			withUser:  true,
			callSave:  true,
			wantCode:  http.StatusOK,
			wantAlias: "test_alias",
		},
		{
			name:     "Success with generated alias",
			url:      "https://google.com",
			withUser: true,
			callSave: true,
			wantCode: http.StatusOK,
		},
		{
			name:      "Empty URL",
			url:       "",
			alias:     "some_alias",
			withUser:  true,
			respError: "field URL is a required field",
			wantCode:  http.StatusOK,
		},
		{
			name:      "Invalid URL",
			url:       "some invalid URL",
			alias:     "some_alias",
			withUser:  true,
			respError: "field URL is not a valid URL",
			wantCode:  http.StatusOK,
		},
		{
			name:      "SaveURL error",
			url:       "https://google.com",
			alias:     "test_alias",
			withUser:  true,
			callSave:  true,
			respError: "failed to save url",
			mockError: errors.New("unexpected error"),
			wantCode:  http.StatusOK,
		},
		{
			name:      "Unauthorized",
			url:       "https://google.com",
			alias:     "test_alias",
			withUser:  false,
			respError: "user is not authorized",
			wantCode:  http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewURLSaver(t)

			if tc.callSave {
				urlSaverMock.On("SaveURL", tc.url, mock.AnythingOfType("string"), int64(1)).
					Return(int64(1), tc.mockError).
					Once()
			}

			handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock)

			input := fmt.Sprintf(`{"url": "%s", "alias": "%s"}`, tc.url, tc.alias)

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			if tc.withUser {
				req = req.WithContext(auth.WithUserID(req.Context(), 1))
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.wantCode, rr.Code)

			var resp save.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			require.Equal(t, tc.respError, resp.Error)

			wantStatus := "OK"
			if tc.respError != "" {
				wantStatus = "Error"
			}
			require.Equal(t, wantStatus, resp.Status)

			if tc.wantAlias != "" {
				require.Equal(t, tc.wantAlias, resp.Alias)
			}

			if tc.callSave && tc.alias == "" && tc.respError == "" {
				require.Len(t, resp.Alias, 6)
			}
		})
	}
}
