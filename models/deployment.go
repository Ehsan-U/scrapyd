package models

import "time"

type Deployment struct {
	Id        uint      `gorm:"primaryKey" json:"id"`
	ProjectId uint      `gorm:"index" json:"project_id" binding:"required"`
	ServerId  uint      `gorm:"index" json:"server_id" binding:"required"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Project Project `gorm:"foreignKey:ProjectId" json:"project"`
	Server  Server  `gorm:"foreignKey:ServerId" json:"server"`
}
