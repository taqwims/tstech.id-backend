package repository

import (
	"github.com/kotban/backend/internal/model"
	"gorm.io/gorm"
)

type TestimonialRepo struct {
	db *gorm.DB
}

func NewTestimonialRepo(db *gorm.DB) *TestimonialRepo {
	return &TestimonialRepo{db: db}
}

func (r *TestimonialRepo) FindActive() ([]model.Testimonial, error) {
	var testimonials []model.Testimonial
	err := r.db.Where("is_active = ?", true).Order("sort_order ASC, created_at DESC").Find(&testimonials).Error
	return testimonials, err
}

func (r *TestimonialRepo) FindAll() ([]model.Testimonial, error) {
	var testimonials []model.Testimonial
	err := r.db.Order("sort_order ASC, created_at DESC").Find(&testimonials).Error
	return testimonials, err
}

func (r *TestimonialRepo) FindByID(id uint) (*model.Testimonial, error) {
	var t model.Testimonial
	err := r.db.First(&t, id).Error
	return &t, err
}

func (r *TestimonialRepo) Create(t *model.Testimonial) error {
	return r.db.Create(t).Error
}

func (r *TestimonialRepo) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&model.Testimonial{}).Where("id = ?", id).Updates(updates).Error
}

func (r *TestimonialRepo) Delete(id uint) error {
	return r.db.Delete(&model.Testimonial{}, id).Error
}
