package models

type Job struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	ProjectID uint   `json:"project_id" gorm:"not null"`
	VersionID uint   `json:"version_id" gorm:"not null"`
	Status    string `json:"status" gorm:"not null"`
	Spider    string `json:"spider" gorm:"not null"`
	Setting   string `json:"setting"`

	Project Project `json:"project" gorm:"foreignKey:ProjectID"`
	Version Version `json:"version" gorm:"foreignKey:VersionID"`
}
