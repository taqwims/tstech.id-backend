package middleware

import (
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

// CORS returns CORS middleware configured for the frontend origin
func CORS() echo.MiddlewareFunc {
	return echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: []string{
			"http://localhost:3000",
			"https://tstech.id",
			"https://www.tstech.id",
			"https://*.vercel.app",
		},
		AllowMethods: []string{
			echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
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
