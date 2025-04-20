package models

import "time"

type Version struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Image     string    `json:"image" gorm:"not null"`
	Spiders   []string  `json:"spiders" gorm:"serializer:json"`
	ProjectID string    `json:"project_id" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
