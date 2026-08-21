package model

import (
	"time"

	"gorm.io/gorm"
)

type Testimonial struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	ClientName  string         `json:"client_name" gorm:"not null"`
	CompanyName string         `json:"company_name"`
	AvatarURL   string         `json:"avatar_url"`
	LogoURL     string         `json:"logo_url"`
	Rating      int            `json:"rating" gorm:"not null;default:5"` // 1-5
	Content     string         `json:"content" gorm:"type:text;not null"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	SortOrder   int            `json:"sort_order" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
