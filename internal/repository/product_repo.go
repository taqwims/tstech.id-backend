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
	if err := r.db.Create(product).Error; err != nil {
		return err
	}
	if product.IsSaaS && product.IsActive {
		r.syncSaaSProduct(product)
	}
	return nil
}

func (r *productRepo) Update(product *model.Product) error {
	if err := r.db.Save(product).Error; err != nil {
		return err
	}
	if product.IsSaaS && product.IsActive {
		r.syncSaaSProduct(product)
	} else {
		// If IsSaaS is false or IsActive is false, deactivate or delete SaaSProduct
		r.db.Model(&model.SaaSProduct{}).Where("slug = ?", product.Slug).Update("is_active", false)
	}
	return nil
}

func (r *productRepo) Delete(id uint) error {
	var p model.Product
	if err := r.db.First(&p, id).Error; err == nil {
		r.db.Where("slug = ?", p.Slug).Delete(&model.SaaSProduct{})
	}
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
	if plan.ProductID != nil && *plan.ProductID > 0 {
		var p model.Product
		if err := r.db.First(&p, *plan.ProductID).Error; err == nil {
			var saasProd model.SaaSProduct
			if err := r.db.Where("slug = ?", p.Slug).First(&saasProd).Error; err == nil {
				plan.SaaSProductID = saasProd.ID
			}
		}
	}
	return r.db.Create(plan).Error
}

func (r *productRepo) UpdatePlan(plan *model.SaaSPlan) error {
	return r.db.Save(plan).Error
}

func (r *productRepo) DeletePlan(id uint) error {
	return r.db.Delete(&model.SaaSPlan{}, id).Error
}

func (r *productRepo) syncSaaSProduct(product *model.Product) {
	var saasProd model.SaaSProduct
	err := r.db.Where("slug = ?", product.Slug).First(&saasProd).Error
	if err != nil {
		saasProd = model.SaaSProduct{
			Slug:             product.Slug,
			Name:             product.Title,
			Tagline:          product.Tagline,
			Icon:             product.Icon,
			Category:         product.Category,
			SubdomainPattern: product.SubdomainPattern,
			BaseDomain:       product.BaseDomain,
			DemoURL:          product.DemoURL,
			DocURL:           product.DocURL,
			Thumbnail:        product.Thumbnail,
			Images:           product.Images,
			Features:         product.Features,
			TechStack:        product.TechStack,
			Overview:         product.Overview,
			WebhookURL:       product.WebhookURL,
			APISecretKey:     product.APISecretKey,
			IsActive:         product.IsActive,
			IsFeatured:       product.IsFeatured,
			SortOrder:        product.SortOrder,
		}
		_ = r.db.Create(&saasProd)
	} else {
		saasProd.Name = product.Title
		saasProd.Tagline = product.Tagline
		saasProd.Icon = product.Icon
		saasProd.Category = product.Category
		saasProd.SubdomainPattern = product.SubdomainPattern
		saasProd.BaseDomain = product.BaseDomain
		saasProd.DemoURL = product.DemoURL
		saasProd.DocURL = product.DocURL
		saasProd.Thumbnail = product.Thumbnail
		saasProd.Images = product.Images
		saasProd.Features = product.Features
		saasProd.TechStack = product.TechStack
		saasProd.Overview = product.Overview
		saasProd.WebhookURL = product.WebhookURL
		if product.APISecretKey != "" {
			saasProd.APISecretKey = product.APISecretKey
		}
		saasProd.IsActive = product.IsActive
		saasProd.IsFeatured = product.IsFeatured
		saasProd.SortOrder = product.SortOrder
		_ = r.db.Save(&saasProd)
	}

	if saasProd.ID > 0 {
		r.db.Model(&model.SaaSPlan{}).Where("product_id = ?", product.ID).Update("saa_s_product_id", saasProd.ID)
	}
}

