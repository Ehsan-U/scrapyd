package models

type Job struct {
	ID        string `json:"id" gorm:"primaryKey"`
	ProjectID string `json:"project_id" gorm:"not null"`
	VersionID string `json:"version_id" gorm:"not null"`
	Status    string `json:"status" gorm:"not null"`
	Spider    string `json:"spider" gorm:"not null"`
	Setting   string `json:"setting"`

	Project Project `json:"project" gorm:"foreignKey:ProjectID"`
	Version Version `json:"version" gorm:"foreignKey:VersionID"`
}
