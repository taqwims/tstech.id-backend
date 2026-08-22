package model

import (
	"time"

	"gorm.io/gorm"
)

type Category struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Type        string         `json:"type" gorm:"size:50;not null;index"` // "blog" or "portfolio"
	Name        string         `json:"name" gorm:"size:100;not null"`
	Slug        string         `json:"slug" gorm:"size:100;not null;index"`
	Description string         `json:"description" gorm:"type:text"`
	Icon        string         `json:"icon" gorm:"size:50"`
	SortOrder   int            `json:"sort_order" gorm:"default:0"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
