package repository

import (
	"github.com/kotban/backend/internal/model"
	"gorm.io/gorm"
)

type ConsultationRepo struct {
	db *gorm.DB
}

func NewConsultationRepo(db *gorm.DB) *ConsultationRepo {
	return &ConsultationRepo{db: db}
}

func (r *ConsultationRepo) Create(c *model.Consultation) error {
	return r.db.Create(c).Error
}

func (r *ConsultationRepo) FindAll(page, perPage int) ([]model.Consultation, int64, error) {
	var consultations []model.Consultation
	var total int64

	r.db.Model(&model.Consultation{}).Count(&total)

	offset := (page - 1) * perPage
	err := r.db.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&consultations).Error
	return consultations, total, err
}

func (r *ConsultationRepo) FindByID(id uint) (*model.Consultation, error) {
	var consultation model.Consultation
	err := r.db.First(&consultation, id).Error
	return &consultation, err
}

func (r *ConsultationRepo) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.Consultation{}).Where("id = ?", id).Update("status", status).Error
}
