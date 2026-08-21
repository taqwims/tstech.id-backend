package repository

import (
	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type OrderRepo struct {
	db *gorm.DB
}

func NewOrderRepo(db *gorm.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) Create(o *model.Order) error {
	return r.db.Create(o).Error
}

func (r *OrderRepo) FindByOrderNumber(orderNumber string) (*model.Order, error) {
	var order model.Order
	err := r.db.Preload("Payments").Where("order_number = ?", orderNumber).First(&order).Error
	return &order, err
}

func (r *OrderRepo) FindByID(id uint) (*model.Order, error) {
	var order model.Order
	err := r.db.Preload("Payments").First(&order, id).Error
	return &order, err
}

func (r *OrderRepo) FindAll(page, perPage int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	r.db.Model(&model.Order{}).Count(&total)

	offset := (page - 1) * perPage
	err := r.db.Preload("Payments").Order("created_at DESC").Offset(offset).Limit(perPage).Find(&orders).Error
	return orders, total, err
}

func (r *OrderRepo) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Update("status", status).Error
}
