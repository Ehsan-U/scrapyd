package types

type ProjectRequest struct {
	Name string `json:"name" binding:"required"`
}

type VersionRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Image     string `json:"image" binding:"required"`
	Tag       string `json:"tag" binding:"required"`
}

type JobRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	VersionID uint   `json:"version_id" binding:"required"`
	Spider    string `json:"spider" binding:"required"`
	Setting   string `json:"setting"`
}

type ServerRequest struct {
	Name    string `json:"name" gorm:"unique" binding:"required"`
	Address string `json:"address" gorm:"unique" binding:"required"`
}
