package model

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID              uint           `json:"id" gorm:"primarykey"`
	OrderNumber     string         `json:"order_number" gorm:"uniqueIndex;not null"`
	PackageType     string         `json:"package_type" gorm:"not null"` // basic / professional / enterprise
	ServiceCategory string         `json:"service_category" gorm:"not null"`
	ProjectName     string         `json:"project_name" gorm:"not null"`
	ProjectDesc     string         `json:"project_desc" gorm:"type:text"`
	ReferenceURLs   string         `json:"reference_urls" gorm:"type:text"` // JSON array
	DesiredDeadline *time.Time     `json:"desired_deadline"`
	CustomerName    string         `json:"customer_name" gorm:"not null"`
	CompanyName     string         `json:"company_name"`
	CustomerEmail   string         `json:"customer_email" gorm:"not null"`
	CustomerWA      string         `json:"customer_wa" gorm:"not null"`
	TotalAmount     int64          `json:"total_amount"`
	DPAmount        int64          `json:"dp_amount"`
	Status          string         `json:"status" gorm:"default:pending"` // pending / dp_paid / in_progress / completed / cancelled
	AgreedToTerms   bool           `json:"agreed_to_terms" gorm:"not null"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
	Payments        []Payment      `json:"payments" gorm:"foreignKey:OrderID"`
}
