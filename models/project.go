package models

import "time"

type Project struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Versions []Version `json:"versions,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE;"`
}
