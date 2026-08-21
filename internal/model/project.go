package model

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID              uint           `json:"id" gorm:"primarykey"`
	OrderID         *uint          `json:"order_id" gorm:"index"`
	ClientUserID    uint           `json:"client_user_id" gorm:"index;not null"`
	AssignedAdminID *uint          `json:"assigned_admin_id" gorm:"index"`
	Title           string         `json:"title" gorm:"not null"`
	Description     string         `json:"description" gorm:"type:text"`
	Status          string         `json:"status" gorm:"default:briefing"` // briefing / quotation / design / development / testing / revision / completed
	ProgressPercent int            `json:"progress_percent" gorm:"default:0"`
	EstimatedStart  *time.Time     `json:"estimated_start"`
	EstimatedEnd    *time.Time     `json:"estimated_end"`
	ActualStart     *time.Time     `json:"actual_start"`
	ActualEnd       *time.Time     `json:"actual_end"`
	TotalAmount     int64          `json:"total_amount"`
	PaidAmount      int64          `json:"paid_amount"`
	Notes           string         `json:"notes" gorm:"type:text"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Client     User        `json:"client" gorm:"foreignKey:ClientUserID"`
	Admin      *User       `json:"admin,omitempty" gorm:"foreignKey:AssignedAdminID"`
	Order      *Order      `json:"order,omitempty" gorm:"foreignKey:OrderID"`
	Milestones []Milestone `json:"milestones,omitempty" gorm:"foreignKey:ProjectID"`
	Payments   []Payment   `json:"payments,omitempty" gorm:"foreignKey:ProjectID"`
}

type Milestone struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	ProjectID   uint       `json:"project_id" gorm:"index;not null"`
	Title       string     `json:"title" gorm:"not null"`
	Description string     `json:"description" gorm:"type:text"`
	Status      string     `json:"status" gorm:"default:pending"` // pending / in_progress / completed
	DueDate     *time.Time `json:"due_date"`
	CompletedAt *time.Time `json:"completed_at"`
	SortOrder   int        `json:"sort_order" gorm:"default:0"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ProjectComment struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	ProjectID uint      `json:"project_id" gorm:"index;not null"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	Type      string    `json:"type" gorm:"default:comment"` // comment / revision_request / status_update
	CreatedAt time.Time `json:"created_at"`
}

type ProjectFile struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	ProjectID  uint      `json:"project_id" gorm:"index;not null"`
	UploadedBy uint      `json:"uploaded_by"`
	Uploader   User      `json:"uploader" gorm:"foreignKey:UploadedBy"`
	FileName   string    `json:"file_name" gorm:"not null"`
	FilePath   string    `json:"file_path" gorm:"not null"`
	FileSize   int64     `json:"file_size"`
	FileType   string    `json:"file_type"`
	Category   string    `json:"category" gorm:"default:document"` // asset / deliverable / document
	CreatedAt  time.Time `json:"created_at"`
}
