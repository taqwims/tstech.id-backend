package handler

import (
	"net/http"

	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
	"github.com/tstech/backend/internal/service"
	"github.com/tstech/backend/pkg/response"
	"github.com/labstack/echo/v4"
)

type ContactHandler struct {
	repo     *repository.ContactRepo
	emailSvc *service.EmailService
}

func NewContactHandler(repo *repository.ContactRepo, emailSvc *service.EmailService) *ContactHandler {
	return &ContactHandler{repo: repo, emailSvc: emailSvc}
}

type CreateContactRequest struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required"`
	Subject string `json:"subject" validate:"required"`
	Message string `json:"message" validate:"required"`
}

func (h *ContactHandler) Create(c echo.Context) error {
	var req CreateContactRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Name == "" || req.Email == "" || req.Subject == "" || req.Message == "" {
		return response.ValidationError(c, map[string]string{
			"message": "Semua field wajib diisi",
		})
	}

	contact := &model.Contact{
		Name:    req.Name,
		Email:   req.Email,
		Subject: req.Subject,
		Message: req.Message,
	}

	if err := h.repo.Create(contact); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengirim pesan")
	}

	// Notify admin
	go h.emailSvc.SendAdminNotification(
		"Pesan Kontak Baru: "+req.Subject,
		"<p>Pesan baru dari "+req.Name+" ("+req.Email+"):</p><p>"+req.Message+"</p>",
	)

	return response.Created(c, contact, "Pesan berhasil dikirim")
}
