package repository

import (
	"strings"

	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type ArticleRepository interface {
	Create(article *model.Article) error
	Update(article *model.Article) error
	Delete(id uint) error
	FindByID(id uint) (*model.Article, error)
	FindBySlug(slug string) (*model.Article, error)
	List(page, perPage int, category, tag, search, status string) ([]model.Article, int64, error)
	GetFeatured(limit int) ([]model.Article, error)
	GetTrending(limit int) ([]model.Article, error)
	GetRelated(category string, excludeID uint, limit int) ([]model.Article, error)
	IncrementViews(id uint) error
	GetCategoriesWithCount() ([]model.ArticleCategoryCount, error)
	GetAllPublished() ([]model.Article, error)
}

type articleRepo struct {
	db *gorm.DB
}

func NewArticleRepo(db *gorm.DB) ArticleRepository {
	return &articleRepo{db: db}
}

func (r *articleRepo) Create(article *model.Article) error {
	return r.db.Create(article).Error
}

func (r *articleRepo) Update(article *model.Article) error {
	return r.db.Save(article).Error
}

func (r *articleRepo) Delete(id uint) error {
	return r.db.Delete(&model.Article{}, id).Error
}

func (r *articleRepo) FindByID(id uint) (*model.Article, error) {
	var article model.Article
	if err := r.db.First(&article, id).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

func (r *articleRepo) FindBySlug(slug string) (*model.Article, error) {
	var article model.Article
	if err := r.db.Where("slug = ?", slug).First(&article).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

func (r *articleRepo) List(page, perPage int, category, tag, search, status string) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	query := r.db.Model(&model.Article{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if category != "" && category != "Semua" {
		query = query.Where("LOWER(category) = LOWER(?)", category)
	}

	if tag != "" {
		query = query.Where("LOWER(tags) LIKE LOWER(?)", "%"+tag+"%")
	}

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&articles).Error
	return articles, total, err
}

func (r *articleRepo) GetFeatured(limit int) ([]model.Article, error) {
	var articles []model.Article
	err := r.db.Where("status = ? AND is_featured = ?", "published", true).
		Order("created_at DESC").
		Limit(limit).
		Find(&articles).Error
	return articles, err
}

func (r *articleRepo) GetTrending(limit int) ([]model.Article, error) {
	var articles []model.Article
	err := r.db.Where("status = ?", "published").
		Order("is_trending DESC, views_count DESC, created_at DESC").
		Limit(limit).
		Find(&articles).Error
	return articles, err
}

func (r *articleRepo) GetRelated(category string, excludeID uint, limit int) ([]model.Article, error) {
	var articles []model.Article
	err := r.db.Where("status = ? AND category = ? AND id != ?", "published", category, excludeID).
		Order("created_at DESC").
		Limit(limit).
		Find(&articles).Error

	// If fewer than limit, fetch other recent published articles
	if len(articles) < limit {
		var additional []model.Article
		excludeIDs := []uint{excludeID}
		for _, a := range articles {
			excludeIDs = append(excludeIDs, a.ID)
		}
		r.db.Where("status = ? AND id NOT IN (?)", "published", excludeIDs).
			Order("views_count DESC, created_at DESC").
			Limit(limit - len(articles)).
			Find(&additional)
		articles = append(articles, additional...)
	}

	return articles, err
}

func (r *articleRepo) IncrementViews(id uint) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("views_count", gorm.Expr("views_count + ?", 1)).Error
}

func (r *articleRepo) GetCategoriesWithCount() ([]model.ArticleCategoryCount, error) {
	var results []model.ArticleCategoryCount
	err := r.db.Model(&model.Article{}).
		Select("category, count(*) as count").
		Where("status = ?", "published").
		Group("category").
		Order("count DESC").
		Scan(&results).Error
	return results, err
}

func (r *articleRepo) GetAllPublished() ([]model.Article, error) {
	var articles []model.Article
	err := r.db.Where("status = ?", "published").
		Order("created_at DESC").
		Find(&articles).Error
	return articles, err
}
