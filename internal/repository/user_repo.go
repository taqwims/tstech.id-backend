package repository

import (
	"time"

	"github.com/kotban/backend/internal/model"
	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepo) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) UpdateLastLogin(id uint, t *time.Time) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("last_login_at", t).Error
}

func (r *UserRepo) List(page, limit int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	r.db.Model(&model.User{}).Count(&total)
	err := r.db.Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&users).Error
	return users, total, err
}

func (r *UserRepo) UpdateProfile(id uint, updates map[string]interface{}) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *UserRepo) SetActive(id uint, active bool) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("is_active", active).Error
}
