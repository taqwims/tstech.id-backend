package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
	"github.com/tstech/backend/internal/service"
	"github.com/tstech/backend/pkg/response"
)

type SaaSHandler struct {
	saasSvc    *service.SaaSService
	saasRepo   repository.SaaSRepository
	storageSvc *service.StorageService
}

func NewSaaSHandler(
	saasSvc *service.SaaSService,
	saasRepo repository.SaaSRepository,
	storageSvc *service.StorageService,
) *SaaSHandler {
	return &SaaSHandler{
		saasSvc:    saasSvc,
		saasRepo:   saasRepo,
		storageSvc: storageSvc,
	}
}

// GET /api/v1/saas/products (Public)
func (h *SaaSHandler) GetProducts(c echo.Context) error {
	products, err := h.saasSvc.GetActiveProducts()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memuat katalog SaaS")
	}
	return response.Success(c, products)
}

// GET /api/v1/saas/products/:slug (Public)
func (h *SaaSHandler) GetProductDetail(c echo.Context) error {
	slug := c.Param("slug")
	product, err := h.saasSvc.GetProductBySlug(slug)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Produk SaaS tidak ditemukan")
	}
	return response.Success(c, product)
}

// GET /api/v1/saas/check-subdomain (Public)
func (h *SaaSHandler) CheckSubdomain(c echo.Context) error {
	productSlug := c.QueryParam("product")
	subdomain := c.QueryParam("subdomain")

	if productSlug == "" || subdomain == "" {
		return response.Error(c, http.StatusBadRequest, "Parameter 'product' dan 'subdomain' wajib disertakan")
	}

	cleanSlug, fullSubdomain, err := h.saasSvc.ValidateSubdomain(productSlug, subdomain)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	return response.Success(c, map[string]interface{}{
		"available":      true,
		"subdomain_slug": cleanSlug,
		"full_subdomain": fullSubdomain,
		"message":        fmt.Sprintf("Subdomain %s tersedia!", fullSubdomain),
	})
}

// GET /api/v1/saas/license/verify (Public / SaaS Satellites)
func (h *SaaSHandler) VerifyLicense(c echo.Context) error {
	domain := c.QueryParam("domain")
	if domain == "" {
		domain = c.QueryParam("subdomain")
	}
	if domain == "" {
		return response.Error(c, http.StatusBadRequest, "Parameter query 'domain' atau 'subdomain' wajib disertakan")
	}

	secretKey := c.Request().Header.Get("X-SaaS-Secret")
	if secretKey == "" {
		secretKey = c.QueryParam("secret")
	}

	licenseInfo, err := h.saasSvc.VerifyLicense(domain, secretKey)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}

	return response.Success(c, licenseInfo)
}

// POST /api/v1/client/saas/subscribe (Client)
func (h *SaaSHandler) Subscribe(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	if userID == 0 {
		return response.Error(c, http.StatusUnauthorized, "Sesi login berakhir, silakan login kembali")
	}

	var req service.SubscribeRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	sub, inv, err := h.saasSvc.Subscribe(userID, req)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	return response.Success(c, map[string]interface{}{
		"subscription": sub,
		"invoice":      inv,
		"message":      "Pemesanan SaaS berhasil dibuat! Silakan lanjutkan pembayaran.",
	})
}

// GET /api/v1/client/saas/subscriptions (Client)
func (h *SaaSHandler) GetUserSubscriptions(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	if userID == 0 {
		return response.Error(c, http.StatusUnauthorized, "Sesi login berakhir")
	}

	subs, err := h.saasSvc.GetUserSubscriptions(userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memuat langganan SaaS")
	}

	return response.Success(c, subs)
}

// GET /api/v1/client/saas/subscriptions/:id (Client)
func (h *SaaSHandler) GetSubscriptionDetail(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	id, _ := strconv.Atoi(c.Param("id"))

	sub, err := h.saasSvc.GetSubscriptionByID(userID, uint(id), false)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Langganan tidak ditemukan")
	}

	return response.Success(c, sub)
}

// POST /api/v1/client/saas/subscriptions/:id/sso-token (Client)
func (h *SaaSHandler) GenerateSSOToken(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	id, _ := strconv.Atoi(c.Param("id"))

	redirectURL, err := h.saasSvc.GenerateSSOToken(userID, uint(id))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	return response.Success(c, map[string]string{
		"redirect_url": redirectURL,
	})
}

// GET /api/v1/client/invoices (Client)
func (h *SaaSHandler) GetUserInvoices(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	if userID == 0 {
		return response.Error(c, http.StatusUnauthorized, "Sesi login berakhir")
	}

	invoices, err := h.saasSvc.GetUserInvoices(userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memuat daftar invoice")
	}

	return response.Success(c, invoices)
}

// GET /api/v1/client/invoices/:id (Client)
func (h *SaaSHandler) GetInvoiceDetail(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	id, _ := strconv.Atoi(c.Param("id"))

	inv, err := h.saasSvc.GetInvoiceByID(userID, uint(id), false)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Invoice tidak ditemukan")
	}

	return response.Success(c, inv)
}

// POST /api/v1/client/invoices/:id/manual-proof (Client)
func (h *SaaSHandler) SubmitManualProof(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	id, _ := strconv.Atoi(c.Param("id"))

	bankName := c.FormValue("bank_name")
	senderName := c.FormValue("sender_name")

	var proofURL string
	file, err := c.FormFile("proof_file")
	if err == nil && file != nil && h.storageSvc != nil {
		res, err := h.storageSvc.SaveFile(file)
		if err == nil && res != nil {
			proofURL = res.FileURL
		}
	}

	if proofURL == "" {
		proofURL = c.FormValue("proof_url")
	}

	if proofURL == "" {
		return response.Error(c, http.StatusBadRequest, "Bukti transfer (foto/file) wajib diunggah")
	}

	inv, err := h.saasSvc.SubmitManualPaymentProof(userID, uint(id), proofURL, bankName, senderName)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	return response.Success(c, inv)
}

// POST /api/v1/webhooks/mayar (Public Webhook)
func (h *SaaSHandler) HandleMayarWebhook(c echo.Context) error {
	var payload struct {
		Event string `json:"event"`
		Data  struct {
			Status        string `json:"status"`
			TransactionID string `json:"id"`
			InvoiceNumber string `json:"extra_data"`
			PaymentLinkID string `json:"payment_link_id"`
		} `json:"data"`
	}

	if err := c.Bind(&payload); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format payload tidak valid")
	}

	_ = h.saasSvc.HandleMayarWebhook(payload.Event, payload.Data.Status, payload.Data.TransactionID, payload.Data.InvoiceNumber)

	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "message": "Webhook processed"})
}

// ADMIN ENDPOINTS

// GET /api/v1/admin/saas/subscriptions
func (h *SaaSHandler) AdminGetAllSubscriptions(c echo.Context) error {
	status := c.QueryParam("status")
	search := c.QueryParam("search")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	subs, total, err := h.saasRepo.GetAllSubscriptions(status, search, page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memuat data langganan SaaS")
	}

	return response.Paginated(c, subs, page, limit, total)
}

// PUT /api/v1/admin/saas/subscriptions/:id
func (h *SaaSHandler) AdminUpdateSubscription(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	sub, err := h.saasRepo.GetSubscriptionByID(uint(id))
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Langganan tidak ditemukan")
	}

	var req struct {
		Status        string `json:"status"`
		AdminNotes    string `json:"admin_notes"`
		CustomDomain  string `json:"custom_domain"`
		ExtendMonths  int    `json:"extend_months"`
		CustomModules string `json:"custom_modules"`
		ActiveUnits   string `json:"active_units"`
		SaaSPlanID    uint   `json:"saas_plan_id"`
	}

	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if req.Status != "" {
		sub.Status = req.Status
	}
	if req.AdminNotes != "" {
		sub.AdminNotes = req.AdminNotes
	}
	if req.CustomDomain != "" {
		sub.CustomDomain = strings.TrimSpace(req.CustomDomain)
	}
	if req.CustomModules != "" {
		sub.CustomModules = req.CustomModules
	}
	if req.ActiveUnits != "" {
		sub.ActiveUnits = req.ActiveUnits
	}
	if req.SaaSPlanID != 0 {
		sub.SaaSPlanID = req.SaaSPlanID
	}
	if req.ExtendMonths > 0 {
		sub.EndDate = sub.EndDate.Add(time.Duration(req.ExtendMonths*30) * 24 * time.Hour)
		sub.Status = "active"
	}

	if err := h.saasRepo.UpdateSubscription(sub); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui langganan")
	}

	return response.Success(c, sub)
}

// POST /api/v1/admin/saas/subscriptions (Admin Onboard Manual Tenant)
func (h *SaaSHandler) AdminCreateSubscription(c echo.Context) error {
	var req struct {
		UserID        uint    `json:"user_id"`
		SaaSProductID uint    `json:"saas_product_id"`
		SaaSPlanID    uint    `json:"saas_plan_id"`
		TenantName    string  `json:"tenant_name"`
		SubdomainSlug string  `json:"subdomain_slug"`
		BillingCycle  string  `json:"billing_cycle"` // "monthly", "yearly"
		DurationMonth int     `json:"duration_months"`
		PriceAmount   float64 `json:"price_amount"`
		CustomDomain  string  `json:"custom_domain"`
		CustomModules string  `json:"custom_modules"`
		ActiveUnits   string  `json:"active_units"`
		Status        string  `json:"status"` // "active", "trial"
	}

	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if req.TenantName == "" || req.SubdomainSlug == "" || req.SaaSProductID == 0 {
		return response.Error(c, http.StatusBadRequest, "Nama tenant, subdomain, dan produk SaaS wajib diisi")
	}

	prod, err := h.saasRepo.GetProductByID(req.SaaSProductID)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Produk SaaS tidak ditemukan")
	}

	cleanSlug, fullDomain, err := h.saasSvc.ValidateSubdomain(prod.Slug, req.SubdomainSlug)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	now := time.Now()
	months := req.DurationMonth
	if months <= 0 {
		if req.BillingCycle == "yearly" {
			months = 12
		} else {
			months = 1
		}
	}
	endDate := now.Add(time.Duration(months*30) * 24 * time.Hour)

	status := req.Status
	if status == "" {
		status = "active"
	}

	subNum := fmt.Sprintf("SUB-ADM-%s-%04d", now.Format("200601"), time.Now().Unix()%10000)

	sub := &model.SaaSSubscription{
		SubscriptionNumber: subNum,
		UserID:             req.UserID,
		SaaSProductID:      prod.ID,
		SaaSPlanID:         req.SaaSPlanID,
		TenantName:         strings.TrimSpace(req.TenantName),
		SubdomainSlug:      cleanSlug,
		FullSubdomain:      fullDomain,
		CustomDomain:       strings.TrimSpace(req.CustomDomain),
		CustomModules:      req.CustomModules,
		ActiveUnits:        req.ActiveUnits,
		Status:             status,
		BillingCycle:       req.BillingCycle,
		PriceAmount:        req.PriceAmount,
		StartDate:          now,
		EndDate:            endDate,
		AutoRenew:          true,
	}

	if sub.UserID == 0 {
		// default to current admin if no user specified
		adminID, _ := c.Get("user_id").(uint)
		if adminID != 0 {
			sub.UserID = adminID
		} else {
			sub.UserID = 2
		}
	}

	if err := h.saasRepo.CreateSubscription(sub); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal membuat tenant langganan")
	}

	return response.Created(c, sub, "Tenant SaaS berhasil didaftarkan")
}

// DELETE /api/v1/admin/saas/subscriptions/:id
func (h *SaaSHandler) AdminDeleteSubscription(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	sub, err := h.saasRepo.GetSubscriptionByID(uint(id))
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Langganan tidak ditemukan")
	}

	sub.Status = "cancelled"
	_ = h.saasRepo.UpdateSubscription(sub)

	return response.Success(c, map[string]string{"message": "Langganan tenant berhasil dibatalkan/dinonaktifkan"})
}

// GET /api/v1/admin/saas/products
func (h *SaaSHandler) AdminGetAllProducts(c echo.Context) error {
	products, err := h.saasRepo.GetAllProductsAdmin()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memuat katalog SaaS admin")
	}
	return response.Success(c, products)
}

// POST /api/v1/admin/saas/products
func (h *SaaSHandler) AdminCreateProduct(c echo.Context) error {
	var prod model.SaaSProduct
	if err := c.Bind(&prod); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format payload tidak valid")
	}

	if prod.Name == "" || prod.Slug == "" {
		return response.Error(c, http.StatusBadRequest, "Nama dan slug produk wajib diisi")
	}

	if prod.APISecretKey == "" {
		prod.APISecretKey = fmt.Sprintf("sec_%s_%d", prod.Slug, time.Now().Unix())
	}
	if prod.SubdomainPattern == "" {
		prod.SubdomainPattern = fmt.Sprintf("{tenant}.%s.tstech.id", prod.Slug)
	}
	if prod.BaseDomain == "" {
		prod.BaseDomain = fmt.Sprintf("%s.tstech.id", prod.Slug)
	}

	if err := h.saasRepo.CreateProduct(&prod); err != nil {
		return response.Error(c, http.StatusBadRequest, fmt.Sprintf("Gagal membuat produk SaaS: %v", err))
	}

	return response.Created(c, prod, "Produk SaaS berhasil ditambahkan")
}

// PUT /api/v1/admin/saas/products/:id
func (h *SaaSHandler) AdminUpdateProduct(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	prod, err := h.saasRepo.GetProductByID(uint(id))
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Produk SaaS tidak ditemukan")
	}

	var req struct {
		Name             string `json:"name"`
		Tagline          string `json:"tagline"`
		Icon             string `json:"icon"`
		Category         string `json:"category"`
		SubdomainPattern string `json:"subdomain_pattern"`
		BaseDomain       string `json:"base_domain"`
		DemoURL          string `json:"demo_url"`
		DocURL           string `json:"doc_url"`
		Thumbnail        string `json:"thumbnail"`
		Features         string `json:"features"`
		TechStack        string `json:"tech_stack"`
		Overview         string `json:"overview"`
		WebhookURL       string `json:"webhook_url"`
		APISecretKey     string `json:"api_secret_key"`
		IsActive         *bool  `json:"is_active"`
		SortOrder        int    `json:"sort_order"`
	}

	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if req.Name != "" {
		prod.Name = req.Name
	}
	if req.Tagline != "" {
		prod.Tagline = req.Tagline
	}
	if req.Icon != "" {
		prod.Icon = req.Icon
	}
	if req.Category != "" {
		prod.Category = req.Category
	}
	if req.SubdomainPattern != "" {
		prod.SubdomainPattern = req.SubdomainPattern
	}
	if req.BaseDomain != "" {
		prod.BaseDomain = req.BaseDomain
	}
	if req.DemoURL != "" {
		prod.DemoURL = req.DemoURL
	}
	if req.DocURL != "" {
		prod.DocURL = req.DocURL
	}
	if req.Thumbnail != "" {
		prod.Thumbnail = req.Thumbnail
	}
	if req.Features != "" {
		prod.Features = req.Features
	}
	if req.TechStack != "" {
		prod.TechStack = req.TechStack
	}
	if req.Overview != "" {
		prod.Overview = req.Overview
	}
	if req.WebhookURL != "" {
		prod.WebhookURL = req.WebhookURL
	}
	if req.APISecretKey != "" {
		prod.APISecretKey = req.APISecretKey
	}
	if req.IsActive != nil {
		prod.IsActive = *req.IsActive
	}
	prod.SortOrder = req.SortOrder

	if err := h.saasRepo.UpdateProduct(prod); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui produk SaaS")
	}

	return response.Success(c, prod)
}

// DELETE /api/v1/admin/saas/products/:id
func (h *SaaSHandler) AdminDeleteProduct(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.saasRepo.DeleteProduct(uint(id)); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menghapus produk SaaS")
	}
	return response.Success(c, map[string]string{"message": "Produk SaaS berhasil dihapus"})
}

// POST /api/v1/admin/saas/products/:id/plans
func (h *SaaSHandler) AdminCreatePlan(c echo.Context) error {
	prodID, _ := strconv.Atoi(c.Param("id"))
	var plan model.SaaSPlan
	if err := c.Bind(&plan); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format payload tidak valid")
	}

	plan.SaaSProductID = uint(prodID)
	if plan.Name == "" || plan.Code == "" {
		return response.Error(c, http.StatusBadRequest, "Nama paket dan code wajib diisi")
	}

	if err := h.saasRepo.CreatePlan(&plan); err != nil {
		return response.Error(c, http.StatusBadRequest, "Gagal membuat paket harga")
	}

	return response.Created(c, plan, "Paket harga SaaS berhasil ditambahkan")
}

// PUT /api/v1/admin/saas/plans/:id
func (h *SaaSHandler) AdminUpdatePlan(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	plan, err := h.saasRepo.GetPlanByID(uint(id))
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Paket harga tidak ditemukan")
	}

	if err := c.Bind(plan); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if err := h.saasRepo.UpdatePlan(plan); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui paket harga")
	}

	return response.Success(c, plan)
}

// DELETE /api/v1/admin/saas/plans/:id
func (h *SaaSHandler) AdminDeletePlan(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.saasRepo.DeletePlan(uint(id)); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menghapus paket harga")
	}
	return response.Success(c, map[string]string{"message": "Paket harga berhasil dihapus"})
}

// GET /api/v1/admin/saas/stats
func (h *SaaSHandler) AdminGetStats(c echo.Context) error {
	stats, err := h.saasRepo.GetSaaSStats()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memuat statistik SaaS")
	}
	return response.Success(c, stats)
}

// GET /api/v1/admin/invoices
func (h *SaaSHandler) AdminGetAllInvoices(c echo.Context) error {
	status := c.QueryParam("status")
	search := c.QueryParam("search")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	invoices, total, err := h.saasRepo.GetAllInvoices(status, search, page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memuat data invoice")
	}

	return response.Paginated(c, invoices, page, limit, total)
}

// POST /api/v1/admin/invoices/:id/approve
func (h *SaaSHandler) AdminApproveInvoice(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	inv, err := h.saasSvc.ApproveManualInvoice(uint(id))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}
	return response.Success(c, inv)
}
