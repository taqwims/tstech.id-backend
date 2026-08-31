package model

import (
	"time"

	"gorm.io/gorm"
)

type Portfolio struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Slug        string         `json:"slug" gorm:"uniqueIndex;not null"`
	Title       string         `json:"title" gorm:"not null"`
	Category    string         `json:"category" gorm:"not null;index"` // website / mobile-app / sistem-informasi / ui-ux
	ClientName  string         `json:"client_name"`
	Industry    string         `json:"industry"`
	Thumbnail   string         `json:"thumbnail"`
	Images      string         `json:"images" gorm:"type:text"` // JSON array of image URLs
	Problem     string         `json:"problem" gorm:"type:text"`
	Solution    string         `json:"solution" gorm:"type:text"`
	TechStack   string         `json:"tech_stack"` // comma-separated
	Result             string         `json:"result" gorm:"type:text"`
	DemoURL            string         `json:"demo_url"`
	IsFeatured         bool           `json:"is_featured" gorm:"default:false;index"`
	SortOrder          int            `json:"sort_order" gorm:"default:0"`
	TestimonialQuote   string         `json:"testimonial_quote" gorm:"type:text"`
	TestimonialAuthor  string         `json:"testimonial_author"`
	TestimonialRole    string         `json:"testimonial_role"`
	TestimonialCompany string         `json:"testimonial_company"`
	TestimonialAvatar  string         `json:"testimonial_avatar"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}
