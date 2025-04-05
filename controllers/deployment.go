package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"scrapyd/api/errs"
	"scrapyd/api/types"
	"scrapyd/models"
)

type DeploymentRequest struct {
	ProjectId uint `json:"project_id" binding:"required"`
	ServerId  uint `json:"server_id" binding:"required"`
	Version   int  `json:"version"`
}

func DeploymentCreate(c *gin.Context) {
	var request DeploymentRequest

	if err := c.MustBindWith(&request, binding.JSON); err != nil {
		return
	}

	var project models.Project
	if err := models.DB.First(&project, request.ProjectId).Error; err != nil {
		c.Error(errs.ErrProjectNotFound)
		return
	}
	var server models.Server
	if err := models.DB.First(&server, request.ServerId).Error; err != nil {
		c.Error(errs.ErrServerNotFound)
		return
	}

	deployment := models.Deployment{
		ProjectId: request.ProjectId,
		ServerId:  request.ServerId,
	}
	if request.Version == 0 {
		request.Version = 1
	} else {
		request.Version += 1
	}

	if err := models.DB.First(&deployment, "project_id = ? AND server_id = ?", deployment.ProjectId, deployment.ServerId).Error; err == nil {
		// same server/project already had a deployment
		c.Error(errs.ErrDeploymentConflict)
		return
	}

	models.DB.Create(&deployment)
	c.JSON(http.StatusCreated, types.Response{
		Status:  "success",
		Message: "created",
	})
}

func DeploymentList(c *gin.Context) {
	var deployments []models.Deployment

	models.DB.Preload("Project").Preload("Server").Find(&deployments)
	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   deployments,
	})
}

func DeploymentGet(c *gin.Context) {
	var deployment models.Deployment

	id := c.Params.ByName("id")
	if err := models.DB.Preload("Project").Preload("Server").First(&deployment, id).Error; err != nil {
		c.Error(errs.ErrDeploymentNotFound)
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   deployment,
	})
}

func DeploymentDelete(c *gin.Context) {
	var existingDeployment models.Deployment

	id := c.Params.ByName("id")
	if err := models.DB.First(&existingDeployment, id).Error; err != nil {
		c.Error(errs.ErrDeploymentNotFound)
		return
	}

	models.DB.Delete(&existingDeployment)
	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "deleted",
	})
}
