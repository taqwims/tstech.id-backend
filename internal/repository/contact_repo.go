package repository

import (
	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type ContactRepo struct {
	db *gorm.DB
}

func NewContactRepo(db *gorm.DB) *ContactRepo {
	return &ContactRepo{db: db}
}

func (r *ContactRepo) Create(c *model.Contact) error {
	return r.db.Create(c).Error
}

func (r *ContactRepo) FindAll(page, perPage int) ([]model.Contact, int64, error) {
	var contacts []model.Contact
	var total int64

	r.db.Model(&model.Contact{}).Count(&total)

	offset := (page - 1) * perPage
	err := r.db.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&contacts).Error
	return contacts, total, err
}
