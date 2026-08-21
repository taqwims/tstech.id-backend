package model

import (
	"time"

	"gorm.io/gorm"
)

type Payment struct {
	ID            uint           `json:"id" gorm:"primarykey"`
	InvoiceNumber string         `json:"invoice_number" gorm:"index"`
	OrderID       *uint          `json:"order_id" gorm:"index"`
	ProjectID     *uint          `json:"project_id" gorm:"index"`
	IpaymuTransID string         `json:"ipaymu_trans_id"`
	GatewayTransID string        `json:"gateway_trans_id" gorm:"index"`
	PaymentURL    string         `json:"payment_url"`
	Amount        int64          `json:"amount" gorm:"not null"`
	PaymentType   string         `json:"payment_type" gorm:"not null"` // dp / pelunasan / full_payment
	PaymentMethod string         `json:"payment_method"`               // mayar / ipaymu / manual / bank_transfer / qris
	Status        string         `json:"status" gorm:"default:pending"` // pending / waiting_confirmation / success / settlement / expire / cancel
	PaidAt        *time.Time     `json:"paid_at"`
	Notes         string         `json:"notes" gorm:"type:text"`
	ProofURL      string         `json:"proof_url"`
	BankName      string         `json:"bank_name"`
	AccountHolder string         `json:"account_holder"`
	GatewayData   string         `json:"gateway_data" gorm:"type:text"` // JSON raw notification data
	IpaymuData    string         `json:"ipaymu_data" gorm:"type:text"`  // JSON raw notification data for backward compatibility
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Order   *Order   `json:"order,omitempty" gorm:"foreignKey:OrderID"`
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}
