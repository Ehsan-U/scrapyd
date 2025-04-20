package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"scrapyd/api/errs"
	"scrapyd/api/types"
	"scrapyd/models"
	"scrapyd/tasks"
)

func VersionCreate(c *gin.Context) {
	var request types.VersionRequest
	var version models.Version

	if err := c.MustBindWith(&request, binding.JSON); err != nil {
		return
	}
	if err := models.DB.First(&models.Project{}, "id = ?", request.ProjectID).Error; err != nil {
		c.Error(errs.ErrProjectNotFound)
		return
	}
	if err := models.DB.First(&version, "id = ? AND project_id = ?", request.ID, request.ProjectID).Error; err == nil {
		c.Error(errs.ErrVersionConflict)
		return
	}

	version.ID = request.ID
	version.ProjectID = request.ProjectID
	version.Image = request.Image

	if err := tasks.NewTask("inspect:version", version.ID); err != nil {
		c.Error(err)
		return
	}

	models.DB.Create(&version)
	c.JSON(http.StatusCreated, types.Response{
		Status:  "success",
		Message: "created",
	})
}

func VersionList(c *gin.Context) {
	var versions []models.Version

	projectID := c.Params.ByName("project_id")
	if err := models.DB.First(&models.Project{}, "id = ?", projectID).Error; err != nil {
		c.Error(errs.ErrProjectNotFound)
		return
	}

	models.DB.Find(&versions, "project_id = ?", projectID)
	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   versions,
	})
}

func VersionDelete(c *gin.Context) {
	var version models.Version

	projectID := c.Params.ByName("project_id")
	if err := models.DB.First(&models.Project{}, "id = ?", projectID).Error; err != nil {
		c.Error(errs.ErrProjectNotFound)
		return
	}

	id := c.Params.ByName("version_id")
	if rows := models.DB.Delete(&version, "id = ? AND project_id = ?", id, projectID).RowsAffected; rows == 0 {
		c.Error(errs.ErrVersionNotFound)
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "deleted",
	})
}
