package repository

import (
	"strings"

	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type ProductRepository interface {
	Create(product *model.Product) error
	Update(product *model.Product) error
	Delete(id uint) error
	FindByID(id uint) (*model.Product, error)
	FindBySlug(slug string) (*model.Product, error)
	ListActive(category string, featuredOnly bool) ([]model.Product, error)
	ListAdmin(category string, search string) ([]model.Product, error)
	CreatePlan(plan *model.SaaSPlan) error
	UpdatePlan(plan *model.SaaSPlan) error
	DeletePlan(id uint) error
}

type productRepo struct {
	db *gorm.DB
}

func NewProductRepo(db *gorm.DB) ProductRepository {
	return &productRepo{db: db}
}

func (r *productRepo) Create(product *model.Product) error {
	return r.db.Create(product).Error
}

func (r *productRepo) Update(product *model.Product) error {
	return r.db.Save(product).Error
}

func (r *productRepo) Delete(id uint) error {
	// Delete associated plans as well
	r.db.Where("product_id = ?", id).Delete(&model.SaaSPlan{})
	return r.db.Delete(&model.Product{}, id).Error
}

func (r *productRepo) FindByID(id uint) (*model.Product, error) {
	var p model.Product
	if err := r.db.Preload("Plans", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order asc, id asc")
	}).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *productRepo) FindBySlug(slug string) (*model.Product, error) {
	var p model.Product
	if err := r.db.Preload("Plans", func(db *gorm.DB) *gorm.DB {
		return db.Where("is_active = ?", true).Order("sort_order asc, id asc")
	}).Where("slug = ?", slug).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *productRepo) ListActive(category string, featuredOnly bool) ([]model.Product, error) {
	var list []model.Product
	query := r.db.Preload("Plans", func(db *gorm.DB) *gorm.DB {
		return db.Where("is_active = ?", true).Order("sort_order asc, id asc")
	}).Where("is_active = ?", true)

	if category != "" && category != "Semua" && category != "all" {
		query = query.Where("LOWER(category) = ?", strings.ToLower(category))
	}

	if featuredOnly {
		query = query.Where("is_featured = ?", true)
	}

	err := query.Order("sort_order ASC, id DESC").Find(&list).Error
	return list, err
}

func (r *productRepo) ListAdmin(category string, search string) ([]model.Product, error) {
	var list []model.Product
	query := r.db.Model(&model.Product{}).Preload("Plans", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order asc, id asc")
	})

	if category != "" && category != "Semua" && category != "all" {
		query = query.Where("LOWER(category) = ?", strings.ToLower(category))
	}

	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(tagline) LIKE ? OR LOWER(tech_stack) LIKE ?", s, s, s)
	}

	err := query.Order("sort_order ASC, id DESC").Find(&list).Error
	return list, err
}

func (r *productRepo) CreatePlan(plan *model.SaaSPlan) error {
	return r.db.Create(plan).Error
}

func (r *productRepo) UpdatePlan(plan *model.SaaSPlan) error {
	return r.db.Save(plan).Error
}

func (r *productRepo) DeletePlan(id uint) error {
	return r.db.Delete(&model.SaaSPlan{}, id).Error
}
