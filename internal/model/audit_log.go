package model

import (
	"time"
)

type AuditLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      *uint     `json:"user_id,omitempty"`
	UserEmail   string    `gorm:"size:100" json:"user_email"`
	UserName    string    `gorm:"size:100" json:"user_name"`
	Role        string    `gorm:"size:50" json:"role"` // admin, client, system, guest
	Action      string    `gorm:"size:100;index" json:"action"` // e.g. AUTH_LOGIN, PAYMENT_VERIFY, QUOTATION_BYPASS, etc.
	Entity      string    `gorm:"size:50;index" json:"entity"` // auth, order, payment, project, quotation, cms, article, ai_blog
	EntityID    string    `gorm:"size:100" json:"entity_id"`
	Description string    `gorm:"type:text" json:"description"`
	Details     string    `gorm:"type:text" json:"details"` // JSON string or extra context
	IPAddress   string    `gorm:"size:60" json:"ip_address"`
	UserAgent   string    `gorm:"size:255" json:"user_agent"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}
