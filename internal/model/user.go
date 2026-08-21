package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Email       string         `json:"email" gorm:"uniqueIndex;not null"`
	Password    string         `json:"-" gorm:"not null"`
	Name        string         `json:"name" gorm:"not null"`
	Role        string         `json:"role" gorm:"not null;default:client"` // admin / client
	Phone       string         `json:"phone"`
	CompanyName string         `json:"company_name"`
	AvatarURL   string         `json:"avatar_url"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	LastLoginAt *time.Time     `json:"last_login_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
