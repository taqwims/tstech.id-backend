package model

import (
	"time"

	"gorm.io/gorm"
)

type Contact struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	Name      string         `json:"name" gorm:"not null"`
	Email     string         `json:"email" gorm:"not null"`
	Subject   string         `json:"subject" gorm:"not null"`
	Message   string         `json:"message" gorm:"type:text;not null"`
	Status    string         `json:"status" gorm:"default:unread"` // unread / read / replied
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
