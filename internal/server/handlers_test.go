package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"shorty/internal/config"
	resp "shorty/internal/pkg/api/response"
	"shorty/internal/server/mocks"
	"shorty/internal/storage"
	"testing"
)

func TestSaveHandler(t *testing.T) {
	tests := map[string]struct {
		alias        string
		input        string
		wantErr      error
		expectedResp Response
		prepare      func(mockUrlProvider *mocks.MockUrlProvider)
	}{
		"Success: generated alias": {
			input: `{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}`,
			expectedResp: Response{
				Response: resp.OK(),
				Alias:    "55555", //length
			},
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().SaveURL(gomock.Any(), gomock.Any()).Return(int64(6), nil)
			},
		},
		"Success: custom alias": {
			alias: "youtb",
			input: `{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "alias": "youtb"}`,
			expectedResp: Response{
				Response: resp.OK(),
				Alias:    "youtb",
			},
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().SaveURL(gomock.Any(), gomock.Any()).Return(int64(8), nil)
			},
		},
		"Empty URL": {
			input:   `{"alias": "55555"}`,
			wantErr: errors.New("\"URL\" field is mandatory"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().SaveURL(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		"Failed to save url": {
			input:   `{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}`,
			wantErr: errors.New("failed to save url"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().SaveURL(gomock.Any(), gomock.Any()).Return(int64(0), errors.New("cannot prepare sql statement"))
			},
		},
		"Url already exists": {
			input:   `{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}`,
			wantErr: errors.New("url already exists"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().SaveURL(gomock.Any(), gomock.Any()).Return(int64(0), fmt.Errorf("%s: %w", "storage.sqlite.SaveURL", storage.ErrURLAlreadyExists))
			},
		},
		"Empty request": {
			wantErr: errors.New("empty request"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().SaveURL(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			//TODO: t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStorage := mocks.NewMockUrlProvider(ctrl)
			tc.prepare(mockStorage)
			handler := SetupRouter(mockStorage, config.Config{}, slog.Default())
			var req *http.Request
			req = httptest.NewRequest(http.MethodPost, "/v1/url", bytes.NewReader([]byte(tc.input)))
			req.SetBasicAuth("", "")

			//response recorder to capture a response from the handler
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			require.Equal(t, w.Code, http.StatusOK)
			b := w.Body.String()

			var response Response
			require.NoError(t, json.Unmarshal([]byte(b), &response))

			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr.Error(), response.Error)
			} else {
				assert.Equal(t, tc.expectedResp.Status, response.Status)
				assert.Equal(t, len(tc.expectedResp.Alias), len(response.Alias))

				if tc.alias != "" {
					assert.Equal(t, tc.expectedResp.Alias, response.Alias)
				}
			}
		})
	}
}

func TestRedirectHandler(t *testing.T) {
	tests := map[string]struct {
		alias    string
		url      string
		wantErr  error
		wantCode int
		prepare  func(mockUrlProvider *mocks.MockUrlProvider)
	}{
		"Success": {
			alias:    "youtb",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			wantCode: http.StatusFound,
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().GetURL(gomock.Any()).Return("https://www.youtube.com/watch?v=dQw4w9WgXcQ", nil)
			},
		},
		"Url does not exist": {
			alias:    "youtb",
			wantCode: http.StatusOK,
			wantErr:  errors.New("url not found for given alias"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().GetURL(gomock.Any()).Return("", storage.ErrURLNotFound)
			},
		},
		"Internal error": {
			alias:    "youtb",
			wantCode: http.StatusOK,
			wantErr:  errors.New("internal error"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().GetURL(gomock.Any()).Return("", errors.New("unexpected error"))
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mocks.NewMockUrlProvider(ctrl)
			tc.prepare(mockStorage)

			r := &router{
				storage: mockStorage,
				log:     slog.Default(),
			}

			chiRouter := chi.NewRouter()
			chiRouter.Get("/v1/{alias}", r.redirectHandler)

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/%s", tc.alias), nil)
			w := httptest.NewRecorder()
			chiRouter.ServeHTTP(w, req)
			require.Equal(t, tc.wantCode, w.Code)

			if tc.wantErr != nil {
				b := w.Body.String()
				var response Response
				require.NoError(t, json.Unmarshal([]byte(b), &response))
				assert.Equal(t, tc.wantErr.Error(), response.Error)
			} else {
				assert.Equal(t, tc.url, w.Header().Get("Location"))
			}
		})
	}
}

func TestUpdateHandler(t *testing.T) {
	tests := map[string]struct {
		oldAlias     string
		newAlias     string
		input        string
		expectedResp Response
		wantErr      error
		prepare      func(mockUrlProvider *mocks.MockUrlProvider)
	}{
		"Successfully updated alias": {
			oldAlias: "youtb",
			newAlias: "qwert",
			input:    `{"new_alias": "qwert"}`,
			expectedResp: Response{
				Response: resp.OK(),
				Alias:    "qwert",
			},
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().UpdateAlias(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		"Empty request": {
			oldAlias: "youtb",
			wantErr:  errors.New("empty request"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().UpdateAlias(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		"Missed mandatory field: new_alias": {
			oldAlias: "youtb",
			input:    `{}`,
			wantErr:  errors.New("\"NewAlias\" field is mandatory"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().UpdateAlias(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		"new_alias is too short": {
			oldAlias: "youtb",
			input:    `{"new_alias": "qw"}`,
			wantErr:  errors.New("invalid request: new alias is too short"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().UpdateAlias(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		"Same alias": {
			oldAlias: "youtb",
			input:    `{"new_alias": "youtb"}`,
			wantErr:  errors.New("new alias is the same as the old one"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().UpdateAlias(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		"Internal error": {
			oldAlias: "youtb",
			input:    `{"new_alias": "qwert"}`,
			wantErr:  errors.New("internal error"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().UpdateAlias(gomock.Any(), gomock.Any()).Return(errors.New("unexpected error"))
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mocks.NewMockUrlProvider(ctrl)
			tc.prepare(mockStorage)
			r := SetupRouter(mockStorage, config.Config{}, slog.Default())
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/url/%s", tc.oldAlias), bytes.NewReader([]byte(tc.input)))
			req.SetBasicAuth("", "")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, w.Code, http.StatusOK)
			b := w.Body.String()

			var response Response
			require.NoError(t, json.Unmarshal([]byte(b), &response))

			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr.Error(), response.Error)
			} else {
				assert.Equal(t, tc.expectedResp.Status, response.Status)
				assert.Equal(t, tc.expectedResp, response)
			}
		})
	}
}
