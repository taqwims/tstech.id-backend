package repository

import (
	"github.com/kotban/backend/internal/model"
	"gorm.io/gorm"
)

type QuotationRepo struct {
	db *gorm.DB
}

func NewQuotationRepo(db *gorm.DB) *QuotationRepo {
	return &QuotationRepo{db: db}
}

func (r *QuotationRepo) Create(quotation *model.Quotation) error {
	return r.db.Create(quotation).Error
}

func (r *QuotationRepo) FindByProjectID(projectID uint) (*model.Quotation, error) {
	var q model.Quotation
	err := r.db.Where("project_id = ?", projectID).Order("created_at DESC").First(&q).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *QuotationRepo) FindByID(id uint) (*model.Quotation, error) {
	var q model.Quotation
	err := r.db.First(&q, id).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *QuotationRepo) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&model.Quotation{}).Where("id = ?", id).Updates(updates).Error
}
