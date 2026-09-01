package handler

import (
	"net/http"

	"github.com/tstech/backend/internal/service"
	"github.com/tstech/backend/pkg/response"
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

// POST /api/v1/auth/firebase (Login with Firebase / Google)
func (h *AuthHandler) FirebaseLogin(c echo.Context) error {
	var req struct {
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Phone     string `json:"phone"`
	}

	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format payload tidak valid")
	}

	if req.Email == "" {
		return response.Error(c, http.StatusBadRequest, "Email akun Google wajib disertakan")
	}

	user, tokens, err := h.authSvc.FirebaseLogin(req.Email, req.Name, req.AvatarURL, req.Phone)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}

	return response.SuccessWithMessage(c, map[string]interface{}{
		"user":   user,
		"tokens": tokens,
	}, "Login Google berhasil")
}

// POST /api/v1/auth/sso/verify (Satellite SaaS SSO Verification)
func (h *AuthHandler) VerifySSO(c echo.Context) error {
	var req struct {
		Token     string `json:"token"`
		SecretKey string `json:"secret_key"`
	}

	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format payload tidak valid")
	}

	token := req.Token
	if token == "" {
		token = c.QueryParam("token")
	}
	if token == "" {
		return response.Error(c, http.StatusBadRequest, "Parameter 'token' wajib disertakan")
	}

	secret := req.SecretKey
	if secret == "" {
		secret = c.Request().Header.Get("X-SaaS-Secret")
	}

	result, err := h.authSvc.VerifySSOToken(token, secret)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}

	return response.Success(c, result)
}
