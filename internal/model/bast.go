package model

import (
	"time"
)

type Bast struct {
	ID               uint      `json:"id" gorm:"primarykey"`
	ProjectID        uint      `json:"project_id" gorm:"index;not null"`
	BastNumber       string    `json:"bast_number" gorm:"index;not null"`
	HandoverDate     time.Time `json:"handover_date"`
	HandoverDay      string    `json:"handover_day"`
	HandoverLocation string    `json:"handover_location"`
	Party1Name       string    `json:"party1_name"`
	Party1Title      string    `json:"party1_title"`
	Party1Company    string    `json:"party1_company"`
	Party1Address    string    `json:"party1_address"`
	Party2Name       string    `json:"party2_name"`
	Party2Title      string    `json:"party2_title"`
	Party2Company    string    `json:"party2_company"`
	Party2Address    string    `json:"party2_address"`
	ProjectTitle     string    `json:"project_title"`
	ContractRef      string    `json:"contract_ref"`
	WarrantyPeriod   string    `json:"warranty_period"`
	AdditionalNotes  string    `json:"additional_notes" gorm:"type:text"`
	Deliverables     string    `json:"deliverables" gorm:"type:text"` // JSON string of deliverables array
	Status           string    `json:"status" gorm:"default:signed"`  // draft / signed / verified
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}
