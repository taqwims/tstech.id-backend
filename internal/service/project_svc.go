package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kotban/backend/internal/model"
	"github.com/kotban/backend/internal/repository"
	"gorm.io/gorm"
)

type ProjectService struct {
	db            *gorm.DB
	projectRepo   *repository.ProjectRepo
	milestoneRepo *repository.MilestoneRepo
	commentRepo   *repository.CommentRepo
	fileRepo      *repository.FileRepo
	quotationRepo *repository.QuotationRepo
	orderRepo     *repository.OrderRepo
}

func NewProjectService(
	db *gorm.DB,
	pRepo *repository.ProjectRepo,
	mRepo *repository.MilestoneRepo,
	cRepo *repository.CommentRepo,
	fRepo *repository.FileRepo,
	qRepo *repository.QuotationRepo,
	oRepo *repository.OrderRepo,
) *ProjectService {
	return &ProjectService{
		db:            db,
		projectRepo:   pRepo,
		milestoneRepo: mRepo,
		commentRepo:   cRepo,
		fileRepo:      fRepo,
		quotationRepo: qRepo,
		orderRepo:     oRepo,
	}
}

type CreateProjectInput struct {
	OrderID         *uint      `json:"order_id"`
	ClientUserID    uint       `json:"client_user_id"`
	AssignedAdminID *uint      `json:"assigned_admin_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Status          string     `json:"status"`
	EstimatedStart  *time.Time `json:"estimated_start"`
	EstimatedEnd    *time.Time `json:"estimated_end"`
	TotalAmount     int64      `json:"total_amount"`
	PaidAmount      int64      `json:"paid_amount"`
	Notes           string     `json:"notes"`
}

func (s *ProjectService) CreateProject(input CreateProjectInput) (*model.Project, error) {
	if input.Title == "" || input.ClientUserID == 0 {
		return nil, errors.New("judul proyek dan ID klien wajib diisi")
	}

	status := input.Status
	if status == "" {
		status = "briefing"
	}

	project := &model.Project{
		OrderID:         input.OrderID,
		ClientUserID:    input.ClientUserID,
		AssignedAdminID: input.AssignedAdminID,
		Title:           input.Title,
		Description:     input.Description,
		Status:          status,
		ProgressPercent: 0,
		EstimatedStart:  input.EstimatedStart,
		EstimatedEnd:    input.EstimatedEnd,
		TotalAmount:     input.TotalAmount,
		PaidAmount:      input.PaidAmount,
		Notes:           input.Notes,
	}

	if err := s.projectRepo.Create(project); err != nil {
		return nil, err
	}

	// Create default milestones
	defaultMilestones := []string{
		"1. Briefing & Requirement Gathering",
		"2. Desain UI/UX & Wireframe",
		"3. Frontend & Backend Development",
		"4. Quality Assurance & Testing",
		"5. UAT & Revisi Klien",
		"6. Deployment & Serah Terima",
	}

	for i, mTitle := range defaultMilestones {
		m := &model.Milestone{
			ProjectID: project.ID,
			Title:     mTitle,
			Status:    "pending",
			SortOrder: i + 1,
		}
		if i == 0 {
			m.Status = "in_progress"
		}
		s.milestoneRepo.Create(m)
	}

	return s.projectRepo.FindByID(project.ID)
}

func (s *ProjectService) GetProjectDetail(id uint) (*model.Project, error) {
	return s.projectRepo.FindByID(id)
}

func (s *ProjectService) ListProjectsByClient(clientID uint, page, limit int) ([]model.Project, int64, error) {
	return s.projectRepo.ListByClient(clientID, page, limit)
}

func (s *ProjectService) ListAllProjects(page, limit int, status string) ([]model.Project, int64, error) {
	return s.projectRepo.ListAll(page, limit, status)
}

func (s *ProjectService) UpdateProjectStatus(id uint, status string) error {
	return s.projectRepo.UpdateStatus(id, status)
}

func (s *ProjectService) UpdateProjectProgress(id uint, progress int) error {
	if progress < 0 {
		progress = 0
	} else if progress > 100 {
		progress = 100
	}
	return s.projectRepo.UpdateProgress(id, progress)
}

func (s *ProjectService) UpdateProject(id uint, updates map[string]interface{}) error {
	return s.projectRepo.Update(id, updates)
}

// Milestone Management
func (s *ProjectService) AddMilestone(m *model.Milestone) error {
	return s.milestoneRepo.Create(m)
}

func (s *ProjectService) UpdateMilestone(id uint, updates map[string]interface{}) error {
	return s.milestoneRepo.Update(id, updates)
}

func (s *ProjectService) CompleteMilestone(id uint) error {
	return s.milestoneRepo.Complete(id)
}

func (s *ProjectService) DeleteMilestone(id uint) error {
	return s.milestoneRepo.Delete(id)
}

// Comments & Discussion
func (s *ProjectService) AddComment(comment *model.ProjectComment) error {
	return s.commentRepo.Create(comment)
}

func (s *ProjectService) GetComments(projectID uint) ([]model.ProjectComment, error) {
	return s.commentRepo.ListByProject(projectID)
}

// Files
func (s *ProjectService) AddFile(file *model.ProjectFile) error {
	return s.fileRepo.Create(file)
}

func (s *ProjectService) GetFiles(projectID uint) ([]model.ProjectFile, error) {
	return s.fileRepo.ListByProject(projectID)
}

func (s *ProjectService) DeleteFile(id uint) error {
	return s.fileRepo.Delete(id)
}

// Quotations
func (s *ProjectService) SaveQuotation(projectID uint, items interface{}, totalAmount int64, days int, validUntil *time.Time) (*model.Quotation, error) {
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}

	q, _ := s.quotationRepo.FindByProjectID(projectID)
	if q == nil {
		q = &model.Quotation{
			ProjectID:     projectID,
			Items:         string(itemsJSON),
			TotalAmount:   totalAmount,
			EstimatedDays: days,
			ValidUntil:    validUntil,
			Status:        "draft",
		}
		err = s.quotationRepo.Create(q)
	} else {
		err = s.quotationRepo.Update(q.ID, map[string]interface{}{
			"items":          string(itemsJSON),
			"total_amount":   totalAmount,
			"estimated_days": days,
			"valid_until":    validUntil,
			"status":         "draft",
		})
	}

	// Synchronize project total amount if quotation amount changed
	if totalAmount > 0 {
		s.db.Model(&model.Project{}).Where("id = ?", projectID).Update("total_amount", totalAmount)
	}

	return q, err
}

func (s *ProjectService) SendQuotation(quotationID uint) error {
	return s.quotationRepo.Update(quotationID, map[string]interface{}{
		"status": "sent",
	})
}

func (s *ProjectService) RespondQuotation(quotationID uint, status, responseText string) error {
	if status != "accepted" && status != "rejected" && status != "negotiating" {
		return errors.New("status respon tidak valid")
	}
	err := s.quotationRepo.Update(quotationID, map[string]interface{}{
		"status":          status,
		"client_response": responseText,
	})
	if err != nil {
		return err
	}

	if status == "accepted" {
		var q model.Quotation
		if err := s.db.First(&q, quotationID).Error; err == nil {
			s.db.Model(&model.Project{}).Where("id = ?", q.ProjectID).Updates(map[string]interface{}{
				"total_amount": q.TotalAmount,
				"status":       "design",
			})
		}
	}
	return nil
}

func (s *ProjectService) GetQuotationByProject(projectID uint) (*model.Quotation, error) {
	return s.quotationRepo.FindByProjectID(projectID)
}

// Payments & Invoices
func (s *ProjectService) GetProjectPayments(projectID uint, orderID *uint) ([]model.Payment, error) {
	var payments []model.Payment
	query := s.db.Model(&model.Payment{})
	if orderID != nil && *orderID > 0 {
		query = query.Where("project_id = ? OR order_id = ?", projectID, *orderID)
	} else {
		query = query.Where("project_id = ?", projectID)
	}
	err := query.Order("created_at DESC").Find(&payments).Error
	return payments, err
}

func (s *ProjectService) RecordPayment(projectID uint, amount int64, paymentType, paymentMethod, notes, status string) (*model.Payment, error) {
	now := time.Now()
	var count int64
	s.db.Model(&model.Payment{}).Count(&count)
	invNum := fmt.Sprintf("INV/%d%02d/%04d", now.Year(), int(now.Month()), count+1)

	if status == "" {
		status = "success"
	}
	if paymentMethod == "" {
		paymentMethod = "manual"
	}

	var paidAt *time.Time
	if status == "success" || status == "settlement" {
		paidAt = &now
	}

	payment := &model.Payment{
		InvoiceNumber: invNum,
		ProjectID:     &projectID,
		Amount:        amount,
		PaymentType:   paymentType,
		PaymentMethod: paymentMethod,
		Status:        status,
		PaidAt:        paidAt,
		Notes:         notes,
	}

	if err := s.db.Create(payment).Error; err != nil {
		return nil, err
	}

	// Update project paid amount if successful
	if status == "success" || status == "settlement" {
		var project model.Project
		if err := s.db.First(&project, projectID).Error; err == nil {
			project.PaidAmount += amount
			if project.PaidAmount >= project.TotalAmount && project.TotalAmount > 0 {
				project.Status = "completed"
			}
			s.db.Save(&project)
		}
	}

	return payment, nil
}
