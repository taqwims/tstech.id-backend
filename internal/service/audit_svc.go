package service

import (
	"encoding/json"
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
)

type AuditService interface {
	Log(c echo.Context, action, entity, entityID, description string, details interface{})
	LogSystem(action, entity, entityID, description string, details interface{})
	ListLogs(entity, action, search string, page, limit int) ([]model.AuditLog, int64, error)
}

type auditService struct {
	repo repository.AuditRepo
}

func NewAuditService(repo repository.AuditRepo) AuditService {
	return &auditService{repo: repo}
}

func (s *auditService) Log(c echo.Context, action, entity, entityID, description string, details interface{}) {
	var userID *uint
	userEmail := "anonymous"
	userName := "Tamu"
	role := "guest"
	ipAddress := ""
	userAgent := ""

	if c != nil {
		ipAddress = c.RealIP()
		userAgent = c.Request().UserAgent()

		if uid, ok := c.Get("user_id").(uint); ok && uid > 0 {
			userID = &uid
		}
		if email, ok := c.Get("user_email").(string); ok && email != "" {
			userEmail = email
		}
		if name, ok := c.Get("user_name").(string); ok && name != "" {
			userName = name
		}
		if r, ok := c.Get("user_role").(string); ok && r != "" {
			role = r
		}
	}

	detailsStr := ""
	if details != nil {
		if str, ok := details.(string); ok {
			detailsStr = str
		} else {
			bytes, err := json.Marshal(details)
			if err == nil {
				detailsStr = string(bytes)
			}
		}
	}

	entry := &model.AuditLog{
		UserID:      userID,
		UserEmail:   userEmail,
		UserName:    userName,
		Role:        role,
		Action:      action,
		Entity:      entity,
		EntityID:    entityID,
		Description: description,
		Details:     detailsStr,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		CreatedAt:   time.Now(),
	}

	go func(auditEntry *model.AuditLog) {
		if err := s.repo.Create(auditEntry); err != nil {
			log.Printf("⚠️ Failed to write audit log: %v", err)
		}
	}(entry)
}

func (s *auditService) LogSystem(action, entity, entityID, description string, details interface{}) {
	detailsStr := ""
	if details != nil {
		if str, ok := details.(string); ok {
			detailsStr = str
		} else {
			bytes, err := json.Marshal(details)
			if err == nil {
				detailsStr = string(bytes)
			}
		}
	}

	entry := &model.AuditLog{
		UserID:      nil,
		UserEmail:   "system@tstech.id",
		UserName:    "System Worker",
		Role:        "system",
		Action:      action,
		Entity:      entity,
		EntityID:    entityID,
		Description: description,
		Details:     detailsStr,
		IPAddress:   "127.0.0.1",
		UserAgent:   "Go-Worker/TsTech",
		CreatedAt:   time.Now(),
	}

	go func(auditEntry *model.AuditLog) {
		if err := s.repo.Create(auditEntry); err != nil {
			log.Printf("⚠️ Failed to write system audit log: %v", err)
		}
	}(entry)
}

func (s *auditService) ListLogs(entity, action, search string, page, limit int) ([]model.AuditLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.repo.FindAll(entity, action, search, limit, offset)
}
