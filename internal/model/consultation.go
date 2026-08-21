package model

import (
	"time"

	"gorm.io/gorm"
)

type Consultation struct {
	ID                 uint           `json:"id" gorm:"primarykey"`
	Name               string         `json:"name" gorm:"not null"`
	WhatsApp           string         `json:"whatsapp" gorm:"not null"`
	Email              string         `json:"email"`
	BusinessName       string         `json:"business_name"`
	ServiceType        string         `json:"service_type" gorm:"not null"` // website / aplikasi / sistem / lainnya
	BudgetRange        string         `json:"budget_range"`
	Description        string         `json:"description" gorm:"type:text"`
	ConsultationMethod string         `json:"consultation_method" gorm:"not null"` // wa / video / meeting
	PreferredDate      *time.Time     `json:"preferred_date"`
	Status             string         `json:"status" gorm:"default:pending"` // pending / contacted / done
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}
