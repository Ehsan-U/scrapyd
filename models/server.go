package models

import "time"

type Server struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"unique" json:"name"`
	Address   string    `gorm:"unique" json:"address"`
	HostName  string    `json:"hostname"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
