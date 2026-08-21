package handler

import (
	"net/http"

	"github.com/kotban/backend/internal/service"
	"github.com/kotban/backend/pkg/response"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

type RegisterRequest struct {
	Name        string `json:"name" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=6"`
	Phone       string `json:"phone"`
	CompanyName string `json:"company_name"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		return response.ValidationError(c, map[string]string{
			"message": "Nama, email, dan password wajib diisi",
		})
	}

	if len(req.Password) < 6 {
		return response.ValidationError(c, map[string]string{
			"password": "Password minimal 6 karakter",
		})
	}

	user, err := h.authSvc.Register(req.Email, req.Password, req.Name, req.Phone, req.CompanyName)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	_, tokens, err := h.authSvc.Login(req.Email, req.Password)
	if err != nil {
		return response.Created(c, map[string]interface{}{
			"user": user,
		}, "Registrasi berhasil, silakan login")
	}

	return response.Created(c, map[string]interface{}{
		"user":   user,
		"tokens": tokens,
	}, "Registrasi berhasil")
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	user, tokens, err := h.authSvc.Login(req.Email, req.Password)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}

	return response.SuccessWithMessage(c, map[string]interface{}{
		"user":   user,
		"tokens": tokens,
	}, "Login berhasil")
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	var req RefreshRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	tokens, err := h.authSvc.RefreshToken(req.RefreshToken)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}

	return response.SuccessWithMessage(c, map[string]interface{}{
		"tokens": tokens,
	}, "Token berhasil diperbarui")
}

func (h *AuthHandler) Me(c echo.Context) error {
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		return response.Error(c, http.StatusUnauthorized, "Tidak terautentikasi")
	}

	user, err := h.authSvc.GetUserByID(userID)
	if err != nil || user == nil {
		return response.Error(c, http.StatusNotFound, "Pengguna tidak ditemukan")
	}

	return response.Success(c, user)
}
