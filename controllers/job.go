package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"scrapyd/api/errs"
	"scrapyd/api/types"
	"scrapyd/models"
)

func JobCreate(c *gin.Context) {
	var request types.JobRequest

	if err := c.MustBindWith(&request, binding.JSON); err != nil {
		return
	}

	if err := models.DB.First(&models.Project{}, request.ProjectID).Error; err != nil {
		c.Error(errs.ErrProjectNotFound)
		return
	}
	if err := models.DB.First(&models.Version{}, request.VersionID).Error; err != nil {
		c.Error(errs.ErrVersionNotFound)
		return
	}

	job := models.Job{
		ProjectID: request.ProjectID,
		VersionID: request.VersionID,
		Status:    "pending",
		Spider:    request.Spider,
		Setting:   request.Setting,
	}
	models.DB.Create(&job)

	c.JSON(http.StatusCreated, types.Response{
		Status:  "success",
		Message: "created",
	})
}

func JobList(c *gin.Context) {
	var jobs []models.Job

	models.DB.Preload("Project").Preload("Version").Find(&jobs)
	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   jobs,
	})
}

func JobGet(c *gin.Context) {
	var job models.Job

	id := c.Params.ByName("id")
	if err := models.DB.Preload("Project").Preload("Version").First(&job, id).Error; err != nil {
		c.Error(errs.ErrJobNotFound)
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   job,
	})
}

func JobUpdate(c *gin.Context) {
	var existingJob models.Job
	var updateData struct {
		Id     uint   `json:"id" binding:"required"`
		Status string `json:"status" binding:"required"`
	}

	if err := c.MustBindWith(&updateData, binding.JSON); err != nil {
		return
	}

	if err := models.DB.First(&existingJob, updateData.Id).Error; err != nil {
		c.Error(errs.ErrJobNotFound)
		return
	}
	existingJob.Status = updateData.Status

	models.DB.Save(&existingJob)
	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "updated",
	})
}

func JobDelete(c *gin.Context) {
	var job models.Job

	id := c.Params.ByName("id")
	if err := models.DB.First(&job, id).Error; err != nil {
		c.Error(errs.ErrJobNotFound)
		return
	}

	models.DB.Delete(&job)
	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "deleted",
	})
}
