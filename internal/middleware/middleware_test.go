package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/tstech/backend/internal/middleware"
)

func TestCORS(t *testing.T) {
	os.Setenv("APP_URL", "https://kotban.com")

	e := echo.New()
	e.Use(middleware.CORS())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	testCases := []struct {
		name           string
		origin         string
		method         string
		expectedAllow  bool
		expectedStatus int
	}{
		{
			name:           "Vercel Production Domain",
			origin:         "https://kotban-frontend.vercel.app",
			method:         http.MethodGet,
			expectedAllow:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Vercel Preview Domain",
			origin:         "https://kotban-frontend-git-feature-taqwims.vercel.app",
			method:         http.MethodOptions,
			expectedAllow:  true,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Localhost Port 3000",
			origin:         "http://localhost:3000",
			method:         http.MethodGet,
			expectedAllow:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Production Domain tstech.id",
			origin:         "https://tstech.id",
			method:         http.MethodGet,
			expectedAllow:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Dynamic APP_URL kotban.com",
			origin:         "https://kotban.com",
			method:         http.MethodGet,
			expectedAllow:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unauthorized Domain",
			origin:         "https://malicious-site.com",
			method:         http.MethodGet,
			expectedAllow:  false,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/test", nil)
			req.Header.Set(echo.HeaderOrigin, tc.origin)
			if tc.method == http.MethodOptions {
				req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPost)
			}

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)

			allowOrigin := rec.Header().Get(echo.HeaderAccessControlAllowOrigin)
			if tc.expectedAllow {
				assert.Equal(t, tc.origin, allowOrigin)
				assert.Equal(t, "true", rec.Header().Get(echo.HeaderAccessControlAllowCredentials))
			} else {
				assert.Empty(t, allowOrigin)
			}
		})
	}
}
