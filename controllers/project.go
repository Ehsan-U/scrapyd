package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"scrapyd/api/errs"
	"scrapyd/api/types"
	"scrapyd/models"
	"scrapyd/services"
)

func ProjectCreate(c *gin.Context) {
	var request types.ProjectRequest
	var project models.Project

	if err := c.MustBindWith(&request, binding.JSON); err != nil {
		return
	}
	project.ID = request.ID
	if rows := models.DB.Create(&project).RowsAffected; rows == 0 {
		c.Error(errs.ErrProjectConflict)
		return
	}

	c.JSON(http.StatusCreated, types.Response{
		Status:  "success",
		Message: "created",
	})
}

func ProjectList(c *gin.Context) {
	var projects []models.Project

	models.DB.Preload("Versions").Find(&projects)
	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   projects,
	})
}

func ProjectDelete(c *gin.Context) {
	var project models.Project

	id := c.Params.ByName("id")
	if err := models.DB.Preload("Versions").First(&project, "id = ?", id).Error; err != nil {
		c.Error(errs.ErrProjectNotFound)
		return
	}

	// cleanup related stuff like version
	err := services.ProjectCleanup(&project)
	if err != nil {
		c.Error(err)
		return
	}

	models.DB.Delete(&project, "id = ?", id)
	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "deleted",
	})
}
