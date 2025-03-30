package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"scrapyd/api/types"
	"scrapyd/models"
	"scrapyd/services"
)

type Host struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address" binding:"required"`
}

func ServerCreate(c *gin.Context) {
	var host Host

	if err := c.MustBindWith(&host, binding.JSON); err != nil {
		return
	}

	d, err := services.NewDaemon(host.Address)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, types.Response{
			Status:  "error",
			Message: fmt.Sprintf("docker daemon not available on %s", host.Address),
		})
		return
	}
	defer d.Client.Close()

	systemInfo, err := d.GetSystemInfo()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, types.Response{
			Status:  "error",
			Message: "server cannot be added",
		})
		return
	}

	server := models.Server{
		Id:       systemInfo.ID,
		Name:     host.Name,
		Address:  host.Address,
		Status:   "up",
		HostName: systemInfo.Name,
		CPU:      systemInfo.NCPU,
		Memory:   systemInfo.MemTotal,
	}
	result := models.DB.FirstOrCreate(&server)
	if result.RowsAffected == 1 {
		c.JSON(http.StatusCreated, types.Response{
			Status:  "success",
			Message: "created",
		})
	} else {
		c.JSON(http.StatusOK, types.Response{
			Status:  "success",
			Message: "exists",
		})
	}
}

func ServerList(c *gin.Context) {
	var servers []models.Server

	models.DB.Find(&servers)
	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   servers,
	})
}

func ServerUpdate(c *gin.Context) {
	var existingServer models.Server

	id := c.Params.ByName("id")
	err := models.DB.First(&existingServer, "Id = ?", id).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, types.Response{
			Status:  "error",
			Message: "server not found",
		})
		return
	}

	var updateData struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.MustBindWith(&updateData, binding.JSON); err != nil {
		return
	}

	existingServer.Name = updateData.Name
	models.DB.Save(&existingServer)
	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "updated",
	})
}

func ServerGet(c *gin.Context) {
	var existingServer models.Server

	id := c.Params.ByName("id")
	err := models.DB.First(&existingServer, "Id = ?", id).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, types.Response{
			Status:  "error",
			Message: "server not found",
		})
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   existingServer,
	})
}

func ServerDelete(c *gin.Context) {
	var existingServer models.Server

	id := c.Params.ByName("id")
	err := models.DB.First(&existingServer, "Id = ?", id).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, types.Response{
			Status:  "error",
			Message: "server not found",
		})
		return
	}

	models.DB.Delete(&existingServer)
	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "deleted",
	})
}
