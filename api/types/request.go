package types

type ProjectRequest struct {
	Name string `json:"name" binding:"required"`
}

type VersionRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Image     string `json:"image" binding:"required"`
	Tag       string `json:"tag" binding:"required"`
}
