package handler

import (
	"log"
	"net/http"

	"github.com/kotban/backend/internal/model"
	"github.com/kotban/backend/internal/repository"
	"github.com/kotban/backend/internal/service"
	"github.com/kotban/backend/pkg/response"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type OrderHandler struct {
	orderSvc   *service.OrderService
	paymentSvc *service.PaymentService
	emailSvc   *service.EmailService
	userRepo   *repository.UserRepo
}

func NewOrderHandler(
	orderSvc *service.OrderService,
	paymentSvc *service.PaymentService,
	emailSvc *service.EmailService,
	userRepo *repository.UserRepo,
) *OrderHandler {
	return &OrderHandler{
		orderSvc:   orderSvc,
		paymentSvc: paymentSvc,
		emailSvc:   emailSvc,
		userRepo:   userRepo,
	}
}

type CreateOrderRequest struct {
	PackageType     string `json:"package_type" validate:"required"`
	ServiceCategory string `json:"service_category" validate:"required"`
	ProjectName     string `json:"project_name" validate:"required"`
	ProjectDesc     string `json:"project_desc"`
	ReferenceURLs   string `json:"reference_urls"`
	DesiredDeadline string `json:"desired_deadline"`
	CustomerName    string `json:"customer_name" validate:"required"`
	CompanyName     string `json:"company_name"`
	CustomerEmail   string `json:"customer_email" validate:"required"`
	CustomerWA      string `json:"customer_wa" validate:"required"`
	TotalAmount     int64  `json:"total_amount" validate:"required"`
	DPAmount        int64  `json:"dp_amount" validate:"required"`
	AgreedToTerms   bool   `json:"agreed_to_terms" validate:"required"`
}

func (h *OrderHandler) Create(c echo.Context) error {
	var req CreateOrderRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.CustomerName == "" || req.CustomerEmail == "" || req.CustomerWA == "" || req.ProjectName == "" {
		return response.ValidationError(c, map[string]string{
			"message": "Nama, email, WhatsApp, dan nama proyek wajib diisi",
		})
	}

	if !req.AgreedToTerms {
		return response.ValidationError(c, map[string]string{
			"message": "Anda harus menyetujui Syarat & Ketentuan",
		})
	}

	order := &model.Order{
		PackageType:     req.PackageType,
		ServiceCategory: req.ServiceCategory,
		ProjectName:     req.ProjectName,
		ProjectDesc:     req.ProjectDesc,
		ReferenceURLs:   req.ReferenceURLs,
		CustomerName:    req.CustomerName,
		CompanyName:     req.CompanyName,
		CustomerEmail:   req.CustomerEmail,
		CustomerWA:      req.CustomerWA,
		TotalAmount:     req.TotalAmount,
		DPAmount:        req.DPAmount,
		AgreedToTerms:   req.AgreedToTerms,
	}

	if err := h.orderSvc.CreateOrder(order); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal membuat pesanan")
	}

	// Auto check & create client user account if not exists
	isNewAccount := false
	defaultPassword := "kotban123"
	existingUser, err := h.userRepo.FindByEmail(req.CustomerEmail)
	if err != nil || existingUser == nil {
		hashed, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err == nil {
			newUser := &model.User{
				Email:       req.CustomerEmail,
				Password:    string(hashed),
				Name:        req.CustomerName,
				Role:        "client",
				Phone:       req.CustomerWA,
				CompanyName: req.CompanyName,
				IsActive:    true,
			}
			if err := h.userRepo.Create(newUser); err == nil {
				isNewAccount = true
				log.Printf("👤 Auto-created new client account for order #%s: %s", order.OrderNumber, req.CustomerEmail)
			}
		}
	}

	// Send confirmation email with login info if new account
	go h.emailSvc.SendOrderConfirmation(req.CustomerEmail, req.CustomerName, order.OrderNumber, req.TotalAmount, defaultPassword, isNewAccount)

	// Notify admin
	go h.emailSvc.SendAdminNotification(
		"Pesanan Baru #"+order.OrderNumber,
		"<p>Pesanan baru diterima dari "+req.CustomerName+"</p><p>Paket: "+req.PackageType+"</p>",
	)

	return response.Created(c, map[string]interface{}{
		"order":          order,
		"is_new_account": isNewAccount,
		"default_pass":   defaultPassword,
	}, "Pesanan berhasil dibuat")
}

func (h *OrderHandler) GetByOrderNumber(c echo.Context) error {
	orderNumber := c.Param("orderNumber")

	order, err := h.orderSvc.GetByOrderNumber(orderNumber)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Pesanan tidak ditemukan")
	}

	return response.Success(c, order)
}
