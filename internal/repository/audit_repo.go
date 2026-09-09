package repository

import (
	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type AuditRepo interface {
	Create(log *model.AuditLog) error
	FindAll(entity, action, search string, limit, offset int) ([]model.AuditLog, int64, error)
}

type auditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) AuditRepo {
	return &auditRepo{db: db}
}

func (r *auditRepo) Create(log *model.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *auditRepo) FindAll(entity, action, search string, limit, offset int) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	query := r.db.Model(&model.AuditLog{})

	if entity != "" {
		query = query.Where("entity = ?", entity)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("description LIKE ? OR user_email LIKE ? OR user_name LIKE ? OR entity_id LIKE ? OR details LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}
