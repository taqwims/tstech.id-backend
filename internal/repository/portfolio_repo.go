package repository

import (
	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type PortfolioRepo struct {
	db *gorm.DB
}

func NewPortfolioRepo(db *gorm.DB) *PortfolioRepo {
	return &PortfolioRepo{db: db}
}

func (r *PortfolioRepo) FindAll(category string) ([]model.Portfolio, error) {
	var portfolios []model.Portfolio
	query := r.db.Order("sort_order ASC, created_at DESC")

	if category != "" {
		query = query.Where("category = ?", category)
	}

	err := query.Find(&portfolios).Error
	return portfolios, err
}

func (r *PortfolioRepo) FindFeatured(limit int) ([]model.Portfolio, error) {
	var portfolios []model.Portfolio
	err := r.db.Where("is_featured = ?", true).Order("sort_order ASC").Limit(limit).Find(&portfolios).Error
	return portfolios, err
}

func (r *PortfolioRepo) FindBySlug(slug string) (*model.Portfolio, error) {
	var portfolio model.Portfolio
	err := r.db.Where("slug = ?", slug).First(&portfolio).Error
	return &portfolio, err
}

func (r *PortfolioRepo) FindByID(id uint) (*model.Portfolio, error) {
	var portfolio model.Portfolio
	err := r.db.First(&portfolio, id).Error
	return &portfolio, err
}

func (r *PortfolioRepo) Create(p *model.Portfolio) error {
	return r.db.Create(p).Error
}

func (r *PortfolioRepo) Update(p *model.Portfolio) error {
	return r.db.Save(p).Error
}

func (r *PortfolioRepo) Delete(id uint) error {
	return r.db.Delete(&model.Portfolio{}, id).Error
}
