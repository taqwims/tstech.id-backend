package model

import (
	"time"

	"gorm.io/gorm"
)

type Service struct {
	ID              uint           `json:"id" gorm:"primarykey"`
	Slug            string         `json:"slug" gorm:"size:120;uniqueIndex;not null"`
	Title           string         `json:"title" gorm:"size:200;not null"`
	Tagline         string         `json:"tagline" gorm:"type:text"`
	Badge           string         `json:"badge" gorm:"size:100"`
	Icon            string         `json:"icon" gorm:"size:50;default:'🌐'"`
	
	// Overview Section
	OverviewTitle   string         `json:"overview_title" gorm:"size:255"`
	OverviewContent string         `json:"overview_content" gorm:"type:text"`
	OverviewImage   string         `json:"overview_image" gorm:"type:text"`
	
	// Process Section
	ProcessTitle    string         `json:"process_title" gorm:"size:255"`
	ProcessContent  string         `json:"process_content" gorm:"type:text"`
	ProcessImage    string         `json:"process_image" gorm:"type:text"`
	
	// Technologies Section
	TechTitle       string         `json:"tech_title" gorm:"size:255"`
	TechContent     string         `json:"tech_content" gorm:"type:text"` // Formatted markdown or categorized list
	TechImage       string         `json:"tech_image" gorm:"type:text"`
	
	// Call To Action Section
	CtaTitle        string         `json:"cta_title" gorm:"size:255"`
	CtaDescription  string         `json:"cta_description" gorm:"type:text"`
	CtaButtonText   string         `json:"cta_button_text" gorm:"size:100;default:'Konsultasi Gratis Sekarang'"`
	CtaButtonURL    string         `json:"cta_button_url" gorm:"size:255;default:'/konsultasi'"`
	
	// Display & Meta
	SortOrder       int            `json:"sort_order" gorm:"default:0"`
	IsActive        bool           `json:"is_active" gorm:"default:true;index"`
	MetaTitle       string         `json:"meta_title" gorm:"size:255"`
	MetaDescription string         `json:"meta_description" gorm:"type:text"`
	MetaKeywords    string         `json:"meta_keywords" gorm:"size:255"`
	
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}
