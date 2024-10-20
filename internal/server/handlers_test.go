package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
			wantErr: fmt.Errorf("\"URL\" field is mandatory"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().SaveURL(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		"Failed to save url": {
			input:   `{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}`,
			wantErr: fmt.Errorf("failed to save url"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().SaveURL(gomock.Any(), gomock.Any()).Return(int64(0), errors.New("cannot prepare sql statement"))
			},
		},
		"Url already exists": {
			input:   `{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}`,
			wantErr: fmt.Errorf("url already exists"),
			prepare: func(mockUrlProvider *mocks.MockUrlProvider) {
				mockUrlProvider.EXPECT().SaveURL(gomock.Any(), gomock.Any()).Return(int64(0), fmt.Errorf("%s: %w", "storage.sqlite.SaveURL", storage.ErrURLAlreadyExists))
			},
		},
		"EOF": {
			wantErr: fmt.Errorf("empty request"),
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
			cfg := &config.Config{
				HTTPServer: config.HTTPServer{
					User:     "test",
					Password: "qwerty123",
				},
			}
			handler := SetupRouter(mockStorage, *cfg, slog.Default())
			var req *http.Request
			req = httptest.NewRequest(http.MethodPost, "/v1/url", bytes.NewReader([]byte(tc.input)))
			req.SetBasicAuth(cfg.HTTPServer.User, cfg.HTTPServer.Password)

			//response recorder to store response from the handler
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
