package types

type ProjectRequest struct {
	ID string `json:"id" binding:"required"`
}

type VersionRequest struct {
	ID        string `json:"id" binding:"required"`
	ProjectID string `json:"project_id" binding:"required"`
	Image     string `json:"image" binding:"required"`
}

type JobRequest struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id" binding:"required"`
	VersionID string `json:"version_id" binding:"required"`
	Spider    string `json:"spider" binding:"required"`
	Setting   string `json:"setting"`
}
