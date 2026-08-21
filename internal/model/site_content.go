package model

import "time"

type SiteContent struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null"`
	Value     string    `json:"value" gorm:"type:text;not null"`
	Type      string    `json:"type" gorm:"default:text"` // text / html / json / image_url
	Group     string    `json:"group" gorm:"index"`       // hero / about / services / faq / footer etc
	UpdatedAt time.Time `json:"updated_at"`
}
