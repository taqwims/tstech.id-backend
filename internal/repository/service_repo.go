package repository

import (
	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type ServiceRepository interface {
	Create(service *model.Service) error
	Update(service *model.Service) error
	Delete(id uint) error
	FindByID(id uint) (*model.Service, error)
	FindBySlug(slug string) (*model.Service, error)
	ListActive() ([]model.Service, error)
	ListAdmin() ([]model.Service, error)
}

type serviceRepo struct {
	db *gorm.DB
}

func NewServiceRepo(db *gorm.DB) ServiceRepository {
	return &serviceRepo{db: db}
}

func (r *serviceRepo) Create(service *model.Service) error {
	return r.db.Create(service).Error
}

func (r *serviceRepo) Update(service *model.Service) error {
	return r.db.Save(service).Error
}

func (r *serviceRepo) Delete(id uint) error {
	return r.db.Delete(&model.Service{}, id).Error
}

func (r *serviceRepo) FindByID(id uint) (*model.Service, error) {
	var s model.Service
	if err := r.db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *serviceRepo) FindBySlug(slug string) (*model.Service, error) {
	var s model.Service
	if err := r.db.Where("slug = ?", slug).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *serviceRepo) ListActive() ([]model.Service, error) {
	var list []model.Service
	err := r.db.Where("is_active = ?", true).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *serviceRepo) ListAdmin() ([]model.Service, error) {
	var list []model.Service
	err := r.db.Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}
