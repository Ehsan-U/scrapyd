package models

import "time"

type Server struct {
	Id        string `gorm:"primary-key"`
	Name      string `gorm:"unique"`
	Address   string `gorm:"unique"`
	HostName  string
	Status    string
	CPU       int
	Memory    int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
