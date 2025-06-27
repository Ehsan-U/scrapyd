package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"scrapyd/api/errs"
	"scrapyd/api/types"
	"scrapyd/models"
	"scrapyd/services"
	"scrapyd/tasks"
)

func VersionCreate(c *gin.Context) {
	var request types.VersionRequest
	var version models.Version

	if err := c.ShouldBind(&request); err != nil {
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

	imageTar, err := c.FormFile("image_tar")
	if err != nil {
		c.Error(errs.ErrVersionImageTarNotFound)
		return
	}

	file, err := imageTar.Open()
	if err != nil {
		c.Error(errs.ErrVersionImageTarInvalid)
		return
	}
	defer file.Close()

	imageName := services.VersionInit(file)
	if imageName == "" {
		c.Error(errs.ErrVersionImageTarInvalid)
		return
	}

	version.ID = request.ID
	version.ProjectID = request.ProjectID
	version.Image = imageName

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
	if err := models.DB.First(&version, "id = ? AND project_id = ?", id, projectID).Error; err != nil {
		c.Error(errs.ErrVersionNotFound)
		return
	}

	// cleanup the related stuff like image
	if err := services.VersionCleanup(&version); err != nil {
		c.Error(err)
		return
	}

	if err := models.DB.Delete(&version).Error; err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "deleted",
	})
}
