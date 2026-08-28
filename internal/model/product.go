package model

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID            uint           `json:"id" gorm:"primarykey"`
	Slug          string         `json:"slug" gorm:"size:150;uniqueIndex;not null"`
	Title         string         `json:"title" gorm:"size:200;not null"`
	Tagline       string         `json:"tagline" gorm:"type:text"`
	Badge         string         `json:"badge" gorm:"size:100"`
	Category      string         `json:"category" gorm:"size:100;index"` // Point of Sale, ERP & Keuangan, Klinik & Medika, Sekolah & LMS, E-Commerce, dsb.
	Icon          string         `json:"icon" gorm:"size:50;default:'📦'"`
	Thumbnail     string         `json:"thumbnail" gorm:"type:text"`
	Images        string         `json:"images" gorm:"type:text"` // JSON array of screenshot URLs
	Price         float64        `json:"price" gorm:"default:0"`
	PriceDiscount float64        `json:"price_discount" gorm:"default:0"`
	PriceType     string         `json:"price_type" gorm:"size:50;default:'one_time'"` // one_time, monthly, yearly, custom
	DemoURL       string         `json:"demo_url" gorm:"size:255"`
	DocURL        string         `json:"doc_url" gorm:"size:255"`
	Features      string         `json:"features" gorm:"type:text"`   // JSON array or bullet list of features
	TechStack     string         `json:"tech_stack" gorm:"type:text"` // e.g. "Next.js, Go, PostgreSQL, TailwindCSS"
	Overview      string         `json:"overview" gorm:"type:text"`   // Rich Markdown description
	IsFeatured    bool           `json:"is_featured" gorm:"default:false;index"`
	IsActive      bool           `json:"is_active" gorm:"default:true;index"`
	SortOrder     int            `json:"sort_order" gorm:"default:0"`
	MetaTitle     string         `json:"meta_title" gorm:"size:255"`
	MetaDescription string       `json:"meta_description" gorm:"type:text"`
	MetaKeywords  string         `json:"meta_keywords" gorm:"size:255"`

	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}
