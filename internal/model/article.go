package model

import "time"

type Article struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Title           string     `gorm:"size:255;not null" json:"title"`
	Slug            string     `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Excerpt         string     `gorm:"type:text" json:"excerpt"`
	Content         string     `gorm:"type:text;not null" json:"content"`
	CoverImage      string     `gorm:"size:500" json:"cover_image"`
	Category        string     `gorm:"size:100;index;not null" json:"category"`
	Tags            string     `gorm:"size:255" json:"tags"` // Comma-separated or JSON string
	AuthorName      string     `gorm:"size:150;default:'Tim Redaksi TsTech'" json:"author_name"`
	AuthorAvatar    string     `gorm:"size:500" json:"author_avatar"`
	AuthorRole      string     `gorm:"size:100;default:'Tech Writer & Engineer'" json:"author_role"`
	Status          string     `gorm:"size:50;default:'published';index" json:"status"` // published, draft, archived
	IsFeatured      bool       `gorm:"default:false;index" json:"is_featured"`
	IsTrending      bool       `gorm:"default:false;index" json:"is_trending"`
	ViewsCount      int        `gorm:"default:0" json:"views_count"`
	ReadingTime     int        `gorm:"default:5" json:"reading_time"` // in minutes
	MetaTitle       string     `gorm:"size:255" json:"meta_title"`
	MetaDescription string     `gorm:"type:text" json:"meta_description"`
	MetaKeywords    string     `gorm:"size:255" json:"meta_keywords"`
	CanonicalURL    string     `gorm:"size:500" json:"canonical_url"`
	PublishedAt     *time.Time `json:"published_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type ArticleCategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}
