package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
	"github.com/tstech/backend/internal/service"
	"github.com/tstech/backend/pkg/response"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db               *gorm.DB
	projectSvc       *service.ProjectService
	contentSvc       *service.ContentService
	storageSvc       *service.StorageService
	orderRepo        *repository.OrderRepo
	userRepo         *repository.UserRepo
	portfolioRepo    *repository.PortfolioRepo
	testimonialRepo  *repository.TestimonialRepo
	consultationRepo *repository.ConsultationRepo
	contactRepo      *repository.ContactRepo
}

func NewAdminHandler(
	db *gorm.DB,
	projectSvc *service.ProjectService,
	contentSvc *service.ContentService,
	storageSvc *service.StorageService,
	orderRepo *repository.OrderRepo,
	userRepo *repository.UserRepo,
	portfolioRepo *repository.PortfolioRepo,
	testimonialRepo *repository.TestimonialRepo,
	consultationRepo *repository.ConsultationRepo,
	contactRepo *repository.ContactRepo,
) *AdminHandler {
	return &AdminHandler{
		db:               db,
		projectSvc:       projectSvc,
		contentSvc:       contentSvc,
		storageSvc:       storageSvc,
		orderRepo:        orderRepo,
		userRepo:         userRepo,
		portfolioRepo:    portfolioRepo,
		testimonialRepo:  testimonialRepo,
		consultationRepo: consultationRepo,
		contactRepo:      contactRepo,
	}
}

// GET /api/admin/dashboard
func (h *AdminHandler) Dashboard(c echo.Context) error {
	var totalOrders int64
	var totalProjects int64
	var activeProjects int64
	var totalClients int64
	var totalConsultations int64
	var totalContacts int64

	h.db.Model(&model.Order{}).Count(&totalOrders)
	h.db.Model(&model.Project{}).Count(&totalProjects)
	h.db.Model(&model.Project{}).Where("status != ?", "completed").Count(&activeProjects)
	h.db.Model(&model.User{}).Where("role = ?", "client").Count(&totalClients)
	h.db.Model(&model.Consultation{}).Count(&totalConsultations)
	h.db.Model(&model.Contact{}).Count(&totalContacts)

	var revenue int64
	h.db.Model(&model.Payment{}).Where("status = ?", "success").Select("COALESCE(SUM(amount), 0)").Scan(&revenue)

	var recentOrders []model.Order
	h.db.Order("created_at DESC").Limit(5).Find(&recentOrders)

	var recentProjects []model.Project
	h.db.Preload("Client").Order("created_at DESC").Limit(5).Find(&recentProjects)

	return response.Success(c, map[string]interface{}{
		"stats": map[string]interface{}{
			"total_orders":        totalOrders,
			"total_projects":      totalProjects,
			"active_projects":     activeProjects,
			"total_clients":       totalClients,
			"total_revenue":       revenue,
			"total_consultations": totalConsultations,
			"total_contacts":      totalContacts,
		},
		"recent_orders":   recentOrders,
		"recent_projects": recentProjects,
	})
}

// GET /api/admin/projects
func (h *AdminHandler) ListProjects(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}
	status := c.QueryParam("status")

	projects, total, err := h.projectSvc.ListAllProjects(page, limit, status)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar proyek")
	}

	return response.Paginated(c, projects, page, limit, total)
}

// POST /api/admin/projects
func (h *AdminHandler) CreateProject(c echo.Context) error {
	var input service.CreateProjectInput
	if err := c.Bind(&input); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	adminID, _ := c.Get("user_id").(uint)
	input.AssignedAdminID = &adminID

	project, err := h.projectSvc.CreateProject(input)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	return response.Created(c, project, "Proyek berhasil dibuat")
}

// GET /api/admin/projects/:id
func (h *AdminHandler) GetProject(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	project, err := h.projectSvc.GetProjectDetail(uint(id))
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Proyek tidak ditemukan")
	}

	quotation, _ := h.projectSvc.GetQuotationByProject(uint(id))
	comments, _ := h.projectSvc.GetComments(uint(id))
	files, _ := h.projectSvc.GetFiles(uint(id))
	payments, _ := h.projectSvc.GetProjectPayments(uint(id), project.OrderID)

	return response.Success(c, map[string]interface{}{
		"project":   project,
		"quotation": quotation,
		"comments":  comments,
		"files":     files,
		"payments":  payments,
	})
}

// POST /api/admin/projects/:id/payments
func (h *AdminHandler) RecordProjectPayment(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Amount        int64  `json:"amount" validate:"required"`
		PaymentType   string `json:"payment_type"`   // dp / pelunasan / full_payment
		PaymentMethod string `json:"payment_method"` // bank_transfer / ipaymu / manual / qris
		Notes         string `json:"notes"`
		Status        string `json:"status"` // success / pending
	}
	if err := c.Bind(&req); err != nil || req.Amount <= 0 {
		return response.Error(c, http.StatusBadRequest, "Jumlah pembayaran (amount) tidak valid")
	}

	if req.PaymentType == "" {
		req.PaymentType = "pelunasan"
	}
	if req.Status == "" {
		req.Status = "success"
	}

	payment, err := h.projectSvc.RecordPayment(uint(id), req.Amount, req.PaymentType, req.PaymentMethod, req.Notes, req.Status)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mencatat pembayaran")
	}

	return response.Created(c, payment, "Pembayaran berhasil dicatat")
}

// PUT /api/admin/projects/:id
func (h *AdminHandler) UpdateProject(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var updates map[string]interface{}
	if err := c.Bind(&updates); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	delete(updates, "id")
	delete(updates, "created_at")

	if err := h.projectSvc.UpdateProject(uint(id), updates); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui proyek")
	}

	return response.SuccessWithMessage(c, nil, "Proyek berhasil diperbarui")
}

// PUT /api/admin/projects/:id/status
func (h *AdminHandler) UpdateProjectStatus(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status string `json:"status" validate:"required"`
	}
	if err := c.Bind(&req); err != nil || req.Status == "" {
		return response.Error(c, http.StatusBadRequest, "Status wajib diisi")
	}

	if err := h.projectSvc.UpdateProjectStatus(uint(id), req.Status); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui status proyek")
	}

	return response.SuccessWithMessage(c, nil, "Status proyek berhasil diperbarui")
}

// PUT /api/admin/projects/:id/progress
func (h *AdminHandler) UpdateProjectProgress(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Progress int `json:"progress_percent"`
	}
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if err := h.projectSvc.UpdateProjectProgress(uint(id), req.Progress); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui progress proyek")
	}

	return response.SuccessWithMessage(c, nil, "Progress proyek berhasil diperbarui")
}

// POST /api/admin/projects/:id/milestones
func (h *AdminHandler) CreateMilestone(c echo.Context) error {
	projectID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Title       string     `json:"title"`
		Description string     `json:"description"`
		DueDate     *time.Time `json:"due_date"`
		SortOrder   int        `json:"sort_order"`
	}
	if err := c.Bind(&req); err != nil || req.Title == "" {
		return response.Error(c, http.StatusBadRequest, "Judul milestone wajib diisi")
	}

	milestone := &model.Milestone{
		ProjectID:   uint(projectID),
		Title:       req.Title,
		Description: req.Description,
		Status:      "pending",
		DueDate:     req.DueDate,
		SortOrder:   req.SortOrder,
	}

	if err := h.projectSvc.AddMilestone(milestone); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menambahkan milestone")
	}

	return response.Created(c, milestone, "Milestone berhasil ditambahkan")
}

// PUT /api/admin/milestones/:id
func (h *AdminHandler) UpdateMilestone(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var updates map[string]interface{}
	if err := c.Bind(&updates); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if err := h.projectSvc.UpdateMilestone(uint(id), updates); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui milestone")
	}

	return response.SuccessWithMessage(c, nil, "Milestone berhasil diperbarui")
}

// PUT /api/admin/milestones/:id/complete
func (h *AdminHandler) CompleteMilestone(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.projectSvc.CompleteMilestone(uint(id)); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menyelesaikan milestone")
	}

	return response.SuccessWithMessage(c, nil, "Milestone diselesaikan")
}

// DELETE /api/admin/milestones/:id
func (h *AdminHandler) DeleteMilestone(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.projectSvc.DeleteMilestone(uint(id)); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menghapus milestone")
	}

	return response.SuccessWithMessage(c, nil, "Milestone dihapus")
}

// POST /api/admin/projects/:id/quotation
func (h *AdminHandler) SaveQuotation(c echo.Context) error {
	projectID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Items         interface{} `json:"items"`
		TotalAmount   int64       `json:"total_amount"`
		EstimatedDays int         `json:"estimated_days"`
		ValidUntil    *time.Time  `json:"valid_until"`
	}
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	q, err := h.projectSvc.SaveQuotation(uint(projectID), req.Items, req.TotalAmount, req.EstimatedDays, req.ValidUntil)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menyimpan penawaran")
	}

	return response.SuccessWithMessage(c, q, "Penawaran harga berhasil disimpan")
}

// PUT /api/admin/quotations/:id/send
func (h *AdminHandler) SendQuotation(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.projectSvc.SendQuotation(uint(id)); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengirim penawaran")
	}

	return response.SuccessWithMessage(c, nil, "Penawaran berhasil dikirim ke klien")
}

// GET /api/admin/orders
func (h *AdminHandler) ListOrders(c echo.Context) error {
	var orders []model.Order
	if err := h.db.Preload("Payments").Order("created_at DESC").Find(&orders).Error; err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar pesanan")
	}
	return response.Success(c, orders)
}

// PUT /api/admin/orders/:id/status
func (h *AdminHandler) UpdateOrderStatus(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&req); err != nil || req.Status == "" {
		return response.Error(c, http.StatusBadRequest, "Status wajib diisi")
	}

	if err := h.db.Model(&model.Order{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui status pesanan")
	}

	return response.SuccessWithMessage(c, nil, "Status pesanan berhasil diperbarui")
}

// POST /api/admin/orders/:id/convert-to-project
func (h *AdminHandler) ConvertOrderToProject(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var order model.Order
	if err := h.db.First(&order, id).Error; err != nil {
		return response.Error(c, http.StatusNotFound, "Pesanan tidak ditemukan")
	}

	// Check if user already exists
	user, err := h.userRepo.FindByEmail(order.CustomerEmail)
	if err != nil || user == nil {
		// Auto create user for client
		hashed, _ := bcrypt.GenerateFromPassword([]byte("tstech123"), bcrypt.DefaultCost)
		newUser := &model.User{
			Email:       order.CustomerEmail,
			Password:    string(hashed),
			Name:        order.CustomerName,
			Role:        "client",
			Phone:       order.CustomerWA,
			CompanyName: order.CompanyName,
			IsActive:    true,
		}
		if err := h.userRepo.Create(newUser); err != nil {
			return response.Error(c, http.StatusInternalServerError, "Gagal membuat akun klien")
		}
		user = newUser
	}

	adminID, _ := c.Get("user_id").(uint)
	orderIDUint := uint(order.ID)

	project, err := h.projectSvc.CreateProject(service.CreateProjectInput{
		OrderID:         &orderIDUint,
		ClientUserID:    user.ID,
		AssignedAdminID: &adminID,
		Title:           fmt.Sprintf("%s (%s)", order.ProjectName, order.PackageType),
		Description:     order.ProjectDesc,
		Status:          "design",
		TotalAmount:     order.TotalAmount,
		PaidAmount:      order.DPAmount,
		Notes:           fmt.Sprintf("Dikonversi dari pesanan #%s", order.OrderNumber),
	})

	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err.Error())
	}

	// Link existing payments to project
	h.db.Model(&model.Payment{}).Where("order_id = ?", order.ID).Update("project_id", project.ID)

	// If no payment record exists yet, create DP payment record
	var paymentCount int64
	h.db.Model(&model.Payment{}).Where("order_id = ? OR project_id = ?", order.ID, project.ID).Count(&paymentCount)
	if paymentCount == 0 && order.DPAmount > 0 {
		h.projectSvc.RecordPayment(project.ID, order.DPAmount, "dp", "ipaymu", fmt.Sprintf("DP 50%% Pesanan #%s", order.OrderNumber), "success")
	}

	// Auto-create itemized Quotation for project
	quotationItems := []map[string]interface{}{
		{
			"name":        fmt.Sprintf("Paket %s - %s", order.PackageType, order.ServiceCategory),
			"description": order.ProjectDesc,
			"qty":         1,
			"unit":        "Paket",
			"price":       order.TotalAmount,
		},
	}
	validUntil := time.Now().AddDate(0, 1, 0)
	q, _ := h.projectSvc.SaveQuotation(project.ID, quotationItems, order.TotalAmount, 14, &validUntil)
	if q != nil {
		h.projectSvc.RespondQuotation(q.ID, "accepted", "Disetujui dari form pesanan paket")
	}

	// Update order status to in_progress
	h.db.Model(&model.Order{}).Where("id = ?", order.ID).Update("status", "in_progress")

	return response.SuccessWithMessage(c, project, "Pesanan berhasil dikonversi menjadi Proyek!")
}

// GET /api/admin/content
func (h *AdminHandler) ListContent(c echo.Context) error {
	contents, err := h.contentSvc.GetAllRaw()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil konten")
	}
	return response.Success(c, contents)
}

// PUT /api/admin/content
func (h *AdminHandler) UpdateContent(c echo.Context) error {
	var items []model.SiteContent
	if err := c.Bind(&items); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if err := h.contentSvc.BatchUpdate(items); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui konten")
	}

	return response.SuccessWithMessage(c, nil, "Konten berhasil disimpan")
}

// GET /api/admin/users
func (h *AdminHandler) ListUsers(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 50
	}

	users, total, err := h.userRepo.List(page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar pengguna")
	}

	return response.Paginated(c, users, page, limit, total)
}

// PUT /api/admin/users/:id/toggle-status
func (h *AdminHandler) ToggleUserStatus(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	user, err := h.userRepo.FindByID(uint(id))
	if err != nil || user == nil {
		return response.Error(c, http.StatusNotFound, "Pengguna tidak ditemukan")
	}

	if user.Role == "admin" {
		return response.Error(c, http.StatusBadRequest, "Tidak dapat mengubah status admin utama")
	}

	newStatus := !user.IsActive
	if err := h.userRepo.SetActive(uint(id), newStatus); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengubah status pengguna")
	}

	return response.SuccessWithMessage(c, map[string]bool{"is_active": newStatus}, "Status pengguna berhasil diubah")
}

// ==================== PORTFOLIO CRUD ====================

// POST /api/admin/portfolios
func (h *AdminHandler) CreatePortfolio(c echo.Context) error {
	var p model.Portfolio
	if err := c.Bind(&p); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if p.Title == "" {
		return response.Error(c, http.StatusBadRequest, "Judul portofolio wajib diisi")
	}

	if p.Slug == "" {
		p.Slug = strings.ToLower(strings.ReplaceAll(p.Title, " ", "-")) + fmt.Sprintf("-%d", time.Now().Unix())
	}

	if err := h.portfolioRepo.Create(&p); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal membuat portofolio")
	}

	return response.Created(c, p, "Portofolio berhasil dibuat")
}

// PUT /api/admin/portfolios/:id
func (h *AdminHandler) UpdatePortfolio(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	p, err := h.portfolioRepo.FindByID(uint(id))
	if err != nil || p == nil {
		return response.Error(c, http.StatusNotFound, "Portofolio tidak ditemukan")
	}

	if err := c.Bind(p); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	p.ID = uint(id)
	if err := h.portfolioRepo.Update(p); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui portofolio")
	}

	return response.SuccessWithMessage(c, p, "Portofolio berhasil diperbarui")
}

// DELETE /api/admin/portfolios/:id
func (h *AdminHandler) DeletePortfolio(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.portfolioRepo.Delete(uint(id)); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menghapus portofolio")
	}

	return response.SuccessWithMessage(c, nil, "Portofolio berhasil dihapus")
}

// ==================== TESTIMONIAL CRUD ====================

// GET /api/admin/testimonials
func (h *AdminHandler) ListTestimonials(c echo.Context) error {
	testimonials, err := h.testimonialRepo.FindAll()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil testimoni")
	}
	return response.Success(c, testimonials)
}

// POST /api/admin/testimonials
func (h *AdminHandler) CreateTestimonial(c echo.Context) error {
	var t model.Testimonial
	if err := c.Bind(&t); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if t.ClientName == "" || t.Content == "" {
		return response.Error(c, http.StatusBadRequest, "Nama klien dan isi testimoni wajib diisi")
	}

	if t.Rating <= 0 {
		t.Rating = 5
	}
	t.IsActive = true

	if err := h.testimonialRepo.Create(&t); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal membuat testimoni")
	}

	return response.Created(c, t, "Testimoni berhasil ditambahkan")
}

// PUT /api/admin/testimonials/:id
func (h *AdminHandler) UpdateTestimonial(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var updates map[string]interface{}
	if err := c.Bind(&updates); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if err := h.testimonialRepo.Update(uint(id), updates); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui testimoni")
	}

	return response.SuccessWithMessage(c, nil, "Testimoni berhasil diperbarui")
}

// DELETE /api/admin/testimonials/:id
func (h *AdminHandler) DeleteTestimonial(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.testimonialRepo.Delete(uint(id)); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menghapus testimoni")
	}

	return response.SuccessWithMessage(c, nil, "Testimoni berhasil dihapus")
}

// ==================== CONSULTATIONS & CONTACTS ====================

// GET /api/admin/consultations
func (h *AdminHandler) ListConsultations(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 50
	}

	consultations, total, err := h.consultationRepo.FindAll(page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar konsultasi")
	}

	return response.Paginated(c, consultations, page, limit, total)
}

// PUT /api/admin/consultations/:id/status
func (h *AdminHandler) UpdateConsultationStatus(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&req); err != nil || req.Status == "" {
		return response.Error(c, http.StatusBadRequest, "Status wajib diisi")
	}

	if err := h.consultationRepo.UpdateStatus(uint(id), req.Status); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui status konsultasi")
	}

	return response.SuccessWithMessage(c, nil, "Status konsultasi berhasil diperbarui")
}

// GET /api/admin/contacts
func (h *AdminHandler) ListContacts(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 50
	}

	contacts, total, err := h.contactRepo.FindAll(page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil pesan kontak")
	}

	return response.Paginated(c, contacts, page, limit, total)
}

// POST /api/admin/upload
func (h *AdminHandler) UploadFile(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "File tidak ditemukan")
	}

	result, err := h.storageSvc.SaveFile(file)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengunggah file")
	}

	return response.SuccessWithMessage(c, result, "File berhasil diunggah")
}

// POST /api/admin/projects/:id/comments
func (h *AdminHandler) AddProjectComment(c echo.Context) error {
	adminID, _ := c.Get("user_id").(uint)
	projectID, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		Content string `json:"content" validate:"required"`
		Type    string `json:"type"`
	}
	if err := c.Bind(&req); err != nil || req.Content == "" {
		return response.Error(c, http.StatusBadRequest, "Pesan tidak boleh kosong")
	}

	cType := req.Type
	if cType == "" {
		cType = "comment"
	}

	comment := &model.ProjectComment{
		ProjectID: uint(projectID),
		UserID:    adminID,
		Content:   req.Content,
		Type:      cType,
	}

	if err := h.projectSvc.AddComment(comment); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengirim pesan")
	}

	return response.Created(c, comment, "Pesan berhasil dikirim")
}

// POST /api/admin/projects/:id/files
func (h *AdminHandler) UploadProjectFile(c echo.Context) error {
	adminID, _ := c.Get("user_id").(uint)
	projectID, _ := strconv.Atoi(c.Param("id"))

	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "File tidak ditemukan")
	}

	category := c.FormValue("category")
	if category == "" {
		category = "deliverable"
	}

	result, err := h.storageSvc.SaveFile(file)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengunggah file")
	}

	pfile := &model.ProjectFile{
		ProjectID:  uint(projectID),
		UploadedBy: adminID,
		FileName:   result.FileName,
		FilePath:   result.FileURL,
		FileSize:   result.FileSize,
		FileType:   result.FileType,
		Category:   category,
	}

	if err := h.projectSvc.AddFile(pfile); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menyimpan data file")
	}

	return response.Created(c, pfile, "File berhasil diunggah")
}
