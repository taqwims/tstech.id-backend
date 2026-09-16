package model

import "time"

type Quotation struct {
	ID             uint       `json:"id" gorm:"primarykey"`
	ProjectID      uint       `json:"project_id" gorm:"index;not null"`
	Items          string     `json:"items" gorm:"type:text"` // JSON: [{name, desc, price}]
	TotalAmount    int64      `json:"total_amount"`
	EstimatedDays  int        `json:"estimated_days"`
	ValidUntil     *time.Time `json:"valid_until"`
	Status              string     `json:"status" gorm:"default:draft"` // draft / sent / accepted / rejected / negotiating
	ClientResponse      string     `json:"client_response" gorm:"type:text"`
	ProposalURL         string     `json:"proposal_url"`
	ProposalFileName    string     `json:"proposal_file_name"`
	HasMaintenance        bool       `json:"has_maintenance" gorm:"default:false"`
	MaintenanceDuration   string     `json:"maintenance_duration"`
	MaintenancePrice      int64      `json:"maintenance_price" gorm:"default:0"`
	AllowComponentPayment bool       `json:"allow_component_payment" gorm:"default:false"`
	ApprovedBy            string     `json:"approved_by"` // client / admin_bypass
	ApprovedAt            *time.Time `json:"approved_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
