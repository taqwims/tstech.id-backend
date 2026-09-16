package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
	"github.com/tstech/backend/internal/service"
	"github.com/tstech/backend/pkg/response"
	"gorm.io/gorm"
)

type ClientHandler struct {
	db         *gorm.DB
	projectSvc *service.ProjectService
	userRepo   *repository.UserRepo
	storageSvc *service.StorageService
	paymentSvc *service.PaymentService
}

func NewClientHandler(
	db *gorm.DB,
	projectSvc *service.ProjectService,
	userRepo *repository.UserRepo,
	storageSvc *service.StorageService,
	paymentSvc *service.PaymentService,
) *ClientHandler {
	return &ClientHandler{
		db:         db,
		projectSvc: projectSvc,
		userRepo:   userRepo,
		storageSvc: storageSvc,
		paymentSvc: paymentSvc,
	}
}

// GET /api/client/profile
func (h *ClientHandler) GetProfile(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Profil tidak ditemukan")
	}
	return response.Success(c, user)
}

// PUT /api/client/profile
func (h *ClientHandler) UpdateProfile(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	var req struct {
		Name        string `json:"name"`
		Phone       string `json:"phone"`
		CompanyName string `json:"company_name"`
	}
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.CompanyName != "" {
		updates["company_name"] = req.CompanyName
	}

	if err := h.userRepo.UpdateProfile(userID, updates); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui profil")
	}

	return response.SuccessWithMessage(c, nil, "Profil berhasil diperbarui")
}

// GET /api/client/projects
func (h *ClientHandler) ListProjects(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}

	projects, total, err := h.projectSvc.ListProjectsByClient(userID, page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar proyek")
	}

	return response.Paginated(c, projects, page, limit, total)
}

// GET /api/client/orders
func (h *ClientHandler) ListOrders(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return response.Error(c, http.StatusNotFound, "Pengguna tidak ditemukan")
	}

	var orders []model.Order
	query := h.db.Preload("Payments").Where("LOWER(customer_email) = LOWER(?)", user.Email)
	if user.ID > 0 {
		query = h.db.Preload("Payments").Where("LOWER(customer_email) = LOWER(?) OR user_id = ?", user.Email, user.ID)
	}
	if err := query.Order("created_at DESC").Find(&orders).Error; err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar pesanan")
	}

	var projectLinks []struct {
		ID      uint
		OrderID uint
	}
	h.db.Model(&model.Project{}).Where("order_id IS NOT NULL AND deleted_at IS NULL").Select("id, order_id").Scan(&projectLinks)
	orderProjectMap := make(map[uint]uint)
	for _, pl := range projectLinks {
		orderProjectMap[pl.OrderID] = pl.ID
	}

	type ClientOrderResponse struct {
		model.Order
		ProjectID   *uint `json:"project_id,omitempty"`
		IsConverted bool  `json:"is_converted"`
	}

	result := make([]ClientOrderResponse, 0, len(orders))
	for _, o := range orders {
		resp := ClientOrderResponse{Order: o}
		if pID, exists := orderProjectMap[o.ID]; exists {
			resp.ProjectID = &pID
			resp.IsConverted = true
		} else if o.Status == "converted" {
			resp.IsConverted = true
		}
		result = append(result, resp)
	}

	return response.Success(c, result)
}

// GET /api/client/projects/:id
func (h *ClientHandler) GetProject(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	userRole, _ := c.Get("user_role").(string)
	projectID, _ := strconv.Atoi(c.Param("id"))

	project, err := h.projectSvc.GetProjectDetail(uint(projectID))
	if err != nil || project == nil {
		return response.Error(c, http.StatusNotFound, "Proyek tidak ditemukan")
	}

	if userRole != "admin" && project.ClientUserID != userID {
		return response.Error(c, http.StatusForbidden, "Akses ditolak")
	}

	quotation, _ := h.projectSvc.GetQuotationByProject(uint(projectID))
	comments, _ := h.projectSvc.GetComments(uint(projectID))
	files, _ := h.projectSvc.GetFiles(uint(projectID))
	payments, _ := h.projectSvc.GetProjectPayments(uint(projectID), project.OrderID)

	return response.Success(c, map[string]interface{}{
		"project":   project,
		"quotation": quotation,
		"comments":  comments,
		"files":     files,
		"payments":  payments,
	})
}

// POST /api/client/projects/:id/comments
func (h *ClientHandler) AddComment(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	userRole, _ := c.Get("user_role").(string)
	projectID, _ := strconv.Atoi(c.Param("id"))

	project, err := h.projectSvc.GetProjectDetail(uint(projectID))
	if err != nil || project == nil {
		return response.Error(c, http.StatusNotFound, "Proyek tidak ditemukan")
	}

	if userRole != "admin" && project.ClientUserID != userID {
		return response.Error(c, http.StatusForbidden, "Akses ditolak")
	}

	var req struct {
		Content string `json:"content" validate:"required"`
		Type    string `json:"type"` // comment / revision_request
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
		UserID:    userID,
		Content:   req.Content,
		Type:      cType,
	}

	if err := h.projectSvc.AddComment(comment); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengirim pesan")
	}

	return response.Created(c, comment, "Pesan/revisi berhasil dikirim")
}

// POST /api/client/projects/:id/files
func (h *ClientHandler) UploadFile(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	userRole, _ := c.Get("user_role").(string)
	projectID, _ := strconv.Atoi(c.Param("id"))

	project, err := h.projectSvc.GetProjectDetail(uint(projectID))
	if err != nil || project == nil {
		return response.Error(c, http.StatusNotFound, "Proyek tidak ditemukan")
	}

	if userRole != "admin" && project.ClientUserID != userID {
		return response.Error(c, http.StatusForbidden, "Akses ditolak")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "File tidak ditemukan")
	}

	category := c.FormValue("category")
	if category == "" {
		category = "asset"
	}

	folder := "projects"
	if category != "" {
		folder = fmt.Sprintf("projects/%s", category)
	}

	result, err := h.storageSvc.SaveFile(file, folder)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengunggah file")
	}

	pfile := &model.ProjectFile{
		ProjectID:  uint(projectID),
		UploadedBy: userID,
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

// PUT /api/client/projects/:id/quotation/respond
func (h *ClientHandler) RespondQuotation(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	userRole, _ := c.Get("user_role").(string)
	projectID, _ := strconv.Atoi(c.Param("id"))

	project, err := h.projectSvc.GetProjectDetail(uint(projectID))
	if err != nil || project == nil {
		return response.Error(c, http.StatusNotFound, "Proyek tidak ditemukan")
	}

	if userRole != "admin" && project.ClientUserID != userID {
		return response.Error(c, http.StatusForbidden, "Akses ditolak")
	}

	var req struct {
		QuotationID uint   `json:"quotation_id" validate:"required"`
		Status      string `json:"status" validate:"required"` // accepted / rejected / negotiating
		Response    string `json:"response"`
	}
	if err := c.Bind(&req); err != nil || req.Status == "" {
		return response.Error(c, http.StatusBadRequest, "Status tanggapan wajib diisi")
	}

	if err := h.projectSvc.RespondQuotation(req.QuotationID, req.Status, req.Response); err != nil {
		return response.Error(c, http.StatusInternalServerError, err.Error())
	}

	if req.Status == "accepted" {
		h.projectSvc.UpdateProjectStatus(uint(projectID), "design")
	}

	return response.SuccessWithMessage(c, nil, "Tanggapan penawaran berhasil dikirim")
}

// POST /api/client/projects/:id/pay-balance
func (h *ClientHandler) CreateBalancePayment(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)
	userRole, _ := c.Get("user_role").(string)
	projectID, _ := strconv.Atoi(c.Param("id"))

	project, err := h.projectSvc.GetProjectDetail(uint(projectID))
	if err != nil || project == nil {
		return response.Error(c, http.StatusNotFound, "Proyek tidak ditemukan")
	}

	if userRole != "admin" && project.ClientUserID != userID {
		return response.Error(c, http.StatusForbidden, "Akses ditolak")
	}

	user, _ := h.userRepo.FindByID(project.ClientUserID)
	remaining := project.TotalAmount - project.PaidAmount
	if remaining <= 0 {
		return response.Error(c, http.StatusBadRequest, "Tagihan proyek ini sudah lunas")
	}

	var reqBody struct {
		PaymentMethod string `json:"payment_method"`
		Amount        int64  `json:"amount"`
		Notes         string `json:"notes"`
		PaymentType   string `json:"payment_type"`
		ComponentDesc string `json:"component_desc"`
	}
	_ = c.Bind(&reqBody)

	payAmount := remaining
	if reqBody.Amount > 0 && reqBody.Amount <= remaining {
		payAmount = reqBody.Amount
	}

	payType := "pelunasan"
	if payAmount < remaining {
		payType = "termin"
	}
	if reqBody.PaymentType != "" {
		payType = reqBody.PaymentType
	}

	productName := fmt.Sprintf("Pelunasan Proyek: %s", project.Title)
	notes := fmt.Sprintf("Pelunasan sisa tagihan proyek #%d", project.ID)
	if payType == "termin" || reqBody.ComponentDesc != "" {
		if reqBody.ComponentDesc != "" {
			productName = fmt.Sprintf("Pembayaran Komponen: %s (%s)", reqBody.ComponentDesc, project.Title)
			notes = fmt.Sprintf("Pembayaran komponen [%s] untuk proyek #%d", reqBody.ComponentDesc, project.ID)
		} else {
			productName = fmt.Sprintf("Pembayaran Termin: %s", project.Title)
			notes = fmt.Sprintf("Pembayaran termin proyek #%d (Nominal: Rp %d)", project.ID, payAmount)
		}
	}
	if reqBody.Notes != "" {
		notes = reqBody.Notes
	}

	buyerName := "Klien TsTech"
	buyerEmail := "client@tstech.id"
	buyerPhone := "081234567890"

	if user != nil {
		if strings.TrimSpace(user.Name) != "" {
			buyerName = user.Name
		}
		if strings.TrimSpace(user.Email) != "" {
			buyerEmail = user.Email
		}
		if len(strings.TrimSpace(user.Phone)) >= 10 {
			buyerPhone = user.Phone
		}
	}
	if buyerPhone == "081234567890" && project.Order != nil && len(strings.TrimSpace(project.Order.CustomerWA)) >= 10 {
		buyerPhone = project.Order.CustomerWA
	}

	prjIDUint := uint(projectID)
	result, err := h.paymentSvc.CreatePayment(&service.CreatePaymentRequest{
		ProjectID:     &prjIDUint,
		OrderID:       project.OrderID,
		Amount:        payAmount,
		PaymentType:   payType,
		PaymentMethod: reqBody.PaymentMethod,
		BuyerName:     buyerName,
		BuyerEmail:    buyerEmail,
		BuyerPhone:    buyerPhone,
		ProductName:   productName,
		Notes:         notes,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal membuat sesi pembayaran: "+err.Error())
	}

	return response.Success(c, result)
}
