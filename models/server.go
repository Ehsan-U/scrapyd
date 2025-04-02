package models

import "time"

type Server struct {
	Id        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"unique" json:"name"`
	Address   string    `gorm:"unique" json:"address"`
	HostName  string    `json:"hostname"`
	Status    string    `json:"status"`
	CPU       int       `json:"cpu"`
	Memory    int64     `json:"memory"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
