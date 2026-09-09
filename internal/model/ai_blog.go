package model

import (
	"time"
)

type AIBlogSetting struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	IsEnabled      bool       `gorm:"default:false" json:"is_enabled"`
	GeminiAPIKey   string     `gorm:"size:255" json:"gemini_api_key"`
	GeminiModel    string     `gorm:"size:100;default:'gemini-3-flash-preview'" json:"gemini_model"`
	IntervalHours  int        `gorm:"default:24" json:"interval_hours"` // 12, 24, 48, 72, 168
	TargetCategory string     `gorm:"size:100;default:'Teknologi'" json:"target_category"`
	DefaultStatus  string     `gorm:"size:50;default:'published'" json:"default_status"` // published / draft
	AutoSEO        bool       `gorm:"default:true" json:"auto_seo"`
	PromptTemplate string     `gorm:"type:text" json:"prompt_template"`
	LastRunAt      *time.Time `json:"last_run_at"`
	NextRunAt      *time.Time `json:"next_run_at"`
	LastRunStatus  string     `gorm:"size:50;default:'idle'" json:"last_run_status"` // idle, running, success, error
	LastRunMessage string     `gorm:"type:text" json:"last_run_message"`
	TotalGenerated int        `gorm:"default:0" json:"total_generated"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
