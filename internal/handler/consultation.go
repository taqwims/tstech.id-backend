package handler

import (
	"net/http"

	"github.com/kotban/backend/internal/model"
	"github.com/kotban/backend/internal/service"
	"github.com/kotban/backend/pkg/response"
	"github.com/labstack/echo/v4"
)

type ConsultationHandler struct {
	svc      *service.ConsultationService
	emailSvc *service.EmailService
}

func NewConsultationHandler(svc *service.ConsultationService, emailSvc *service.EmailService) *ConsultationHandler {
	return &ConsultationHandler{svc: svc, emailSvc: emailSvc}
}

type CreateConsultationRequest struct {
	Name               string `json:"name" validate:"required"`
	WhatsApp           string `json:"whatsapp" validate:"required"`
	Email              string `json:"email"`
	BusinessName       string `json:"business_name"`
	ServiceType        string `json:"service_type" validate:"required"`
	BudgetRange        string `json:"budget_range"`
	Description        string `json:"description"`
	ConsultationMethod string `json:"consultation_method" validate:"required"`
	PreferredDate      string `json:"preferred_date"`
}

func (h *ConsultationHandler) Create(c echo.Context) error {
	var req CreateConsultationRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Name == "" || req.WhatsApp == "" || req.ServiceType == "" || req.ConsultationMethod == "" {
		return response.ValidationError(c, map[string]string{
			"message": "Nama, WhatsApp, jenis kebutuhan, dan metode konsultasi wajib diisi",
		})
	}

	consultation := &model.Consultation{
		Name:               req.Name,
		WhatsApp:           req.WhatsApp,
		Email:              req.Email,
		BusinessName:       req.BusinessName,
		ServiceType:        req.ServiceType,
		BudgetRange:        req.BudgetRange,
		Description:        req.Description,
		ConsultationMethod: req.ConsultationMethod,
	}

	if err := h.svc.Create(consultation); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menyimpan data konsultasi")
	}

	// Send confirmation email (non-blocking)
	if req.Email != "" {
		go h.emailSvc.SendConsultationConfirmation(req.Email, req.Name)
	}

	// Notify admin
	go h.emailSvc.SendAdminNotification(
		"Konsultasi Baru dari "+req.Name,
		"<p>Konsultasi baru diterima:</p><ul><li>Nama: "+req.Name+"</li><li>WA: "+req.WhatsApp+"</li><li>Jenis: "+req.ServiceType+"</li></ul>",
	)

	return response.Created(c, consultation, "Permintaan konsultasi berhasil dikirim")
}
