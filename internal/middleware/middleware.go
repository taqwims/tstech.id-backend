package middleware

import (
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

// CORS returns CORS middleware configured for the frontend origin
func CORS() echo.MiddlewareFunc {
	return echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOriginFunc: func(origin string) (bool, error) {
			if origin == "" {
				return true, nil
			}
			// Allow localhost / local development
			if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
				return true, nil
			}
			// Allow all Vercel domains (*.vercel.app)
			if strings.HasSuffix(origin, ".vercel.app") {
				return true, nil
			}
			// Allow known production domains and all subdomains (*.tstech.id)
			if origin == "https://tstech.id" || origin == "https://www.tstech.id" ||
				strings.HasSuffix(origin, ".tstech.id") ||
				origin == "https://kotban.com" || origin == "https://www.kotban.com" {
				return true, nil
			}
			// Allow dynamic APP_URL from environment
			appURL := os.Getenv("APP_URL")
			if appURL != "" && (origin == appURL || origin == strings.TrimRight(appURL, "/")) {
				return true, nil
			}
			return false, nil
		},
		AllowMethods: []string{
			echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			"X-Requested-With",
			"Accept-Language",
			"Cache-Control",
			"X-SaaS-Secret",
			"X-Tenant-Slug",
			"X-Tenant-Domain",
			"X-Tenant",
			"X-Subdomain",
		},
		AllowCredentials: true,
	})
}

// Logger returns request logger middleware
func Logger() echo.MiddlewareFunc {
	return echoMiddleware.LoggerWithConfig(echoMiddleware.LoggerConfig{
		Format: "${time_rfc3339} | ${status} | ${latency_human} | ${remote_ip} | ${method} ${uri}\n",
	})
}

// RateLimiter returns rate limiter middleware
func RateLimiter() echo.MiddlewareFunc {
	return echoMiddleware.RateLimiter(echoMiddleware.NewRateLimiterMemoryStore(20))
}

// Recover returns panic recovery middleware
func Recover() echo.MiddlewareFunc {
	return echoMiddleware.Recover()
}
