package model

import "time"

type Quotation struct {
	ID             uint       `json:"id" gorm:"primarykey"`
	ProjectID      uint       `json:"project_id" gorm:"index;not null"`
	Items          string     `json:"items" gorm:"type:text"` // JSON: [{name, desc, price}]
	TotalAmount    int64      `json:"total_amount"`
	EstimatedDays  int        `json:"estimated_days"`
	ValidUntil     *time.Time `json:"valid_until"`
	Status         string     `json:"status" gorm:"default:draft"` // draft / sent / accepted / rejected / negotiating
	ClientResponse string     `json:"client_response" gorm:"type:text"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
