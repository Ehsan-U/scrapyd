package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"scrapyd/api/types"
	"scrapyd/models"
	"scrapyd/services"
)

func DaemonStatus(c *gin.Context) {
	d, err := services.NewDaemon()
	if err != nil {
		c.Error(err)
		return
	}

	info, err := d.GetSystemInfo()
	if err != nil {
		c.Error(err)
		return
	}

	var pendingJobs int64
	models.DB.Model(&models.Job{}).Where("status = ?", "pending").Count(&pendingJobs)
	var runningJobs int64
	models.DB.Model(&models.Job{}).Where("status = ?", "running").Count(&runningJobs)
	var finishedJobs int64
	models.DB.Model(&models.Job{}).Where("status = ?", "finished").Count(&finishedJobs)

	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data: map[string]any{
			"node_name": info.Name,
			"status":    "ok",
			"pending":   pendingJobs,
			"running":   runningJobs,
			"finished":  finishedJobs,
		},
	})
}
