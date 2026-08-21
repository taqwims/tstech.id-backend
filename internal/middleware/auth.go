package middleware

import (
	"net/http"
	"strings"

	"github.com/kotban/backend/internal/service"
	"github.com/kotban/backend/pkg/response"
	"github.com/labstack/echo/v4"
)

// JWTAuth validates the JWT token from Authorization header
func JWTAuth(authSvc *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Error(c, http.StatusUnauthorized, "Token tidak ditemukan")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return response.Error(c, http.StatusUnauthorized, "Format token tidak valid")
			}

			claims, err := authSvc.ValidateToken(parts[1])
			if err != nil {
				return response.Error(c, http.StatusUnauthorized, "Token tidak valid atau sudah kadaluarsa")
			}

			// Set user info in context
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)
			c.Set("user_name", claims.Name)

			return next(c)
		}
	}
}

// RequireRole checks if the user has the required role
func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole, ok := c.Get("user_role").(string)
			if !ok {
				return response.Error(c, http.StatusForbidden, "Akses ditolak")
			}

			for _, role := range roles {
				if userRole == role {
					return next(c)
				}
			}

			return response.Error(c, http.StatusForbidden, "Anda tidak memiliki akses untuk halaman ini")
		}
	}
}
