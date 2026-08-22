package repository

import (
	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type CategoryRepo struct {
	db *gorm.DB
}

func NewCategoryRepo(db *gorm.DB) *CategoryRepo {
	return &CategoryRepo{db: db}
}

// FindAll returns active categories by type, ordered by sort_order
func (r *CategoryRepo) FindAll(catType string) ([]model.Category, error) {
	var categories []model.Category
	q := r.db.Where("is_active = ?", true)
	if catType != "" {
		q = q.Where("type = ?", catType)
	}
	err := q.Order("sort_order ASC, name ASC").Find(&categories).Error
	return categories, err
}

// FindAllAdmin returns all categories by type for admin
func (r *CategoryRepo) FindAllAdmin(catType string) ([]model.Category, error) {
	var categories []model.Category
	q := r.db.Model(&model.Category{})
	if catType != "" {
		q = q.Where("type = ?", catType)
	}
	err := q.Order("type ASC, sort_order ASC, name ASC").Find(&categories).Error
	return categories, err
}

// FindByID returns a category by ID
func (r *CategoryRepo) FindByID(id uint) (*model.Category, error) {
	var category model.Category
	err := r.db.First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// FindBySlug returns a category by type and slug
func (r *CategoryRepo) FindBySlug(catType, slug string) (*model.Category, error) {
	var category model.Category
	err := r.db.Where("type = ? AND slug = ?", catType, slug).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// Create inserts a new category
func (r *CategoryRepo) Create(category *model.Category) error {
	return r.db.Create(category).Error
}

// Update updates an existing category
func (r *CategoryRepo) Update(category *model.Category) error {
	return r.db.Save(category).Error
}

// Delete soft-deletes a category
func (r *CategoryRepo) Delete(id uint) error {
	return r.db.Delete(&model.Category{}, id).Error
}
