package handler

import (
	"net/http"
	"strconv"

	"github.com/kotban/backend/internal/model"
	"github.com/kotban/backend/internal/service"
	"github.com/kotban/backend/pkg/response"
	"github.com/labstack/echo/v4"
)

type PaymentHandler struct {
	paymentSvc *service.PaymentService
	storageSvc *service.StorageService
}

func NewPaymentHandler(paymentSvc *service.PaymentService, storageSvc *service.StorageService) *PaymentHandler {
	return &PaymentHandler{
		paymentSvc: paymentSvc,
		storageSvc: storageSvc,
	}
}

type CreatePaymentAPIRequest struct {
	OrderID       *uint  `json:"order_id"`
	ProjectID     *uint  `json:"project_id"`
	Amount        int64  `json:"amount" validate:"required"`
	PaymentType   string `json:"payment_type" validate:"required"` // dp / pelunasan / full_payment
	PaymentMethod string `json:"payment_method"`                  // mayar / ipaymu / manual
	BuyerName     string `json:"buyer_name" validate:"required"`
	BuyerEmail    string `json:"buyer_email" validate:"required"`
	BuyerPhone    string `json:"buyer_phone" validate:"required"`
	ProductName   string `json:"product_name" validate:"required"`
	Notes         string `json:"notes"`
}

// GetMethods returns active public payment methods configured by admin
func (h *PaymentHandler) GetMethods(c echo.Context) error {
	result, err := h.paymentSvc.GetPublicPaymentMethods()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memuat metode pembayaran: "+err.Error())
	}
	return response.Success(c, result)
}

// Create creates a new payment using configured gateway or manual transfer
func (h *PaymentHandler) Create(c echo.Context) error {
	var req CreatePaymentAPIRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Amount <= 0 {
		return response.ValidationError(c, map[string]string{
			"message": "Jumlah pembayaran (amount) harus lebih dari 0",
		})
	}

	result, err := h.paymentSvc.CreatePayment(&service.CreatePaymentRequest{
		OrderID:       req.OrderID,
		ProjectID:     req.ProjectID,
		Amount:        req.Amount,
		PaymentType:   req.PaymentType,
		PaymentMethod: req.PaymentMethod,
		BuyerName:     req.BuyerName,
		BuyerEmail:    req.BuyerEmail,
		BuyerPhone:    req.BuyerPhone,
		ProductName:   req.ProductName,
		Notes:         req.Notes,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal membuat pembayaran: "+err.Error())
	}

	return response.Success(c, result)
}

// extractWebhookData parses JSON body, URL-encoded Form, and Query params seamlessly
func extractWebhookData(c echo.Context) map[string]interface{} {
	data := make(map[string]interface{})
	_ = c.Bind(&data)

	// Merge form values if present
	if c.Request().Form == nil {
		_ = c.Request().ParseForm()
	}
	for k, v := range c.Request().Form {
		if len(v) > 0 && data[k] == nil {
			data[k] = v[0]
		}
	}

	// Merge query parameters if present
	for k, v := range c.QueryParams() {
		if len(v) > 0 && data[k] == nil {
			data[k] = v[0]
		}
	}

	return data
}

// NotifyMayar handles Mayar.id webhook notifications
func (h *PaymentHandler) NotifyMayar(c echo.Context) error {
	data := extractWebhookData(c)
	if len(data) == 0 {
		return response.Error(c, http.StatusBadRequest, "Empty webhook notification payload")
	}

	if err := h.paymentSvc.HandleMayarNotification(data); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Failed to process Mayar webhook: "+err.Error())
	}

	return response.Success(c, map[string]string{"status": "ok", "provider": "mayar"})
}

// NotifyPakasir handles Pakasir webhook notifications
func (h *PaymentHandler) NotifyPakasir(c echo.Context) error {
	data := extractWebhookData(c)
	if len(data) == 0 {
		return response.Error(c, http.StatusBadRequest, "Empty webhook notification payload")
	}

	if err := h.paymentSvc.HandlePakasirNotification(data); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Failed to process Pakasir notification: "+err.Error())
	}

	return response.Success(c, map[string]string{"status": "ok", "provider": "pakasir"})
}

// Notify handles iPaymu webhook notifications
func (h *PaymentHandler) Notify(c echo.Context) error {
	data := extractWebhookData(c)
	if len(data) == 0 {
		return response.Error(c, http.StatusBadRequest, "Empty webhook notification payload")
	}

	// Check if this is a Pakasir payload (contains 'project' or 'order_id')
	if _, isPakasir := data["project"]; isPakasir {
		if err := h.paymentSvc.HandlePakasirNotification(data); err != nil {
			return response.Error(c, http.StatusInternalServerError, "Failed to process Pakasir notification: "+err.Error())
		}
		return response.Success(c, map[string]string{"status": "ok", "provider": "pakasir"})
	}

	// Check if this is a Mayar payload (contains 'event' or 'messages')
	if _, isMayar := data["event"]; isMayar {
		if err := h.paymentSvc.HandleMayarNotification(data); err != nil {
			return response.Error(c, http.StatusInternalServerError, "Failed to process Mayar webhook")
		}
		return response.Success(c, map[string]string{"status": "ok", "provider": "mayar"})
	}

	if err := h.paymentSvc.HandleIpaymuNotification(data); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Failed to process iPaymu notification: "+err.Error())
	}

	return response.Success(c, map[string]string{"status": "ok", "provider": "ipaymu"})
}

// UploadProof uploads payment transfer receipt for manual verification
func (h *PaymentHandler) UploadProof(c echo.Context) error {
	idParam := c.Param("id")
	paymentID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid payment ID")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "File bukti pembayaran wajib diunggah")
	}

	bankName := c.FormValue("bank_name")
	accountHolder := c.FormValue("account_holder")
	notes := c.FormValue("notes")

	uploadRes, err := h.storageSvc.SaveFile(file)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengunggah file: "+err.Error())
	}

	payment, err := h.paymentSvc.UploadPaymentProof(uint(paymentID), uploadRes.FileURL, bankName, accountHolder, notes)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui bukti pembayaran: "+err.Error())
	}

	return response.SuccessWithMessage(c, payment, "Bukti pembayaran berhasil diunggah. Tim kami akan segera memverifikasinya.")
}

// GetAdminSettings returns gateway configs for admin
func (h *PaymentHandler) GetAdminSettings(c echo.Context) error {
	settings, err := h.paymentSvc.GetAdminSettings()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memuat pengaturan: "+err.Error())
	}
	return response.Success(c, settings)
}

// UpdateAdminSettings saves gateway configs from admin
func (h *PaymentHandler) UpdateAdminSettings(c echo.Context) error {
	var cfg model.PaymentSettingsConfig
	if err := c.Bind(&cfg); err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid request body")
	}

	if err := h.paymentSvc.SaveAdminSettings(&cfg); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menyimpan pengaturan: "+err.Error())
	}

	return response.SuccessWithMessage(c, cfg, "Pengaturan metode pembayaran berhasil disimpan")
}

// ListAdminPayments lists all transactions for admin
func (h *PaymentHandler) ListAdminPayments(c echo.Context) error {
	status := c.QueryParam("status")
	method := c.QueryParam("method")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}

	payments, total, err := h.paymentSvc.ListAllPayments(status, method, page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memuat data pembayaran: "+err.Error())
	}

	return response.Paginated(c, payments, page, limit, total)
}

// VerifyAdminPayment allows admin to approve or reject a manual payment
func (h *PaymentHandler) VerifyAdminPayment(c echo.Context) error {
	idParam := c.Param("id")
	paymentID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid payment ID")
	}

	var req struct {
		Status string `json:"status" validate:"required"` // success / cancel / expire
		Notes  string `json:"notes"`
	}
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Status == "" {
		req.Status = "success"
	}

	payment, err := h.paymentSvc.VerifyManualPayment(uint(paymentID), req.Status, req.Notes)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memverifikasi pembayaran: "+err.Error())
	}

	return response.SuccessWithMessage(c, payment, "Status pembayaran berhasil diperbarui")
}

// VerifyDocument verifies an invoice or quotation document
func (h *PaymentHandler) VerifyDocument(c echo.Context) error {
	docNumber := c.QueryParam("num")
	if docNumber == "" {
		docNumber = c.Param("number")
	}
	docType := c.QueryParam("type")

	if docNumber == "" {
		return response.Error(c, http.StatusBadRequest, "Nomor dokumen wajib disertakan")
	}

	result, err := h.paymentSvc.VerifyDocument(docNumber, docType)
	if err != nil {
		return response.Error(c, http.StatusNotFound, err.Error())
	}

	return response.Success(c, result)
}

// SyncStatus manually or automatically polls the payment gateway to synchronize live status
func (h *PaymentHandler) SyncStatus(c echo.Context) error {
	idParam := c.Param("id")
	paymentID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		// Fallback to identifier lookup
		payment, errSync := h.paymentSvc.SyncPaymentByIdentifier(idParam)
		if errSync != nil {
			return response.Error(c, http.StatusNotFound, "Pembayaran tidak ditemukan: "+errSync.Error())
		}
		return response.SuccessWithMessage(c, payment, "Status pembayaran berhasil disinkronisasi")
	}

	payment, err := h.paymentSvc.SyncPaymentWithGateway(uint(paymentID))
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal sinkronisasi pembayaran: "+err.Error())
	}

	return response.SuccessWithMessage(c, payment, "Status pembayaran berhasil disinkronisasi")
}

// GetStatusAndSync retrieves the payment status and immediately syncs with gateway if pending
func (h *PaymentHandler) GetStatusAndSync(c echo.Context) error {
	identifier := c.Param("identifier")
	if identifier == "" {
		identifier = c.QueryParam("invoice")
	}
	if identifier == "" {
		identifier = c.QueryParam("id")
	}

	payment, err := h.paymentSvc.SyncPaymentByIdentifier(identifier)
	if err != nil {
		return response.Error(c, http.StatusNotFound, err.Error())
	}

	return response.Success(c, payment)
}

