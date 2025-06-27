package types

type ProjectRequest struct {
	ID string `json:"id" binding:"required"`
}

type VersionRequest struct {
	ID        string `form:"id" json:"id" binding:"required"`
	ProjectID string `form:"project_id" json:"project_id" binding:"required"`
}

type JobRequest struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id" binding:"required"`
	VersionID string `json:"version_id" binding:"required"`
	Spider    string `json:"spider" binding:"required"`
	Setting   string `json:"setting"`
}
