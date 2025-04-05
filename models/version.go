package models

import "time"

type Version struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Tag       string    `json:"tag" gorm:"not null"`
	Image     string    `json:"image" gorm:"not null"`
	ProjectID uint      `json:"project_id" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
