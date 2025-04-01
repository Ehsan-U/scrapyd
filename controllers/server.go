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

type Host struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address" binding:"required"`
}

func ServerCreate(c *gin.Context) {
	var host Host
	var server models.Server

	if err := c.MustBindWith(&host, binding.JSON); err != nil {
		return
	}

	// check DB
	if err := models.DB.First(&server, "Address = ?", host.Address).Error; err == nil {
		c.JSON(http.StatusOK, types.Response{
			Status:  "success",
			Message: "exists",
		})
		return
	}

	d, err := services.NewDaemon(host.Address)
	if err != nil {
		c.Error(errs.ErrDaemonConnectionFailed)
		return
	}
	defer d.Client.Close()

	systemInfo, err := d.GetSystemInfo()
	if err != nil {
		c.Error(errs.ErrDaemonNotResponding)
		return
	}

	server.Id = systemInfo.ID
	server.Name = host.Name
	server.Address = host.Address
	server.Status = "up"
	server.HostName = systemInfo.Name
	server.CPU = systemInfo.NCPU
	server.Memory = systemInfo.MemTotal

	if rows := models.DB.Create(&server).RowsAffected; rows == 0 {
		c.JSON(http.StatusCreated, types.Response{
			Status:  "success",
			Message: "exists",
		})
		return
	}
	c.JSON(http.StatusCreated, types.Response{
		Status:  "success",
		Message: "created",
	})
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
		c.Error(errs.ErrServerNotFound)
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
		c.Error(errs.ErrServerNotFound)
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
		c.Error(errs.ErrServerNotFound)
		return
	}

	models.DB.Delete(&existingServer)
	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "deleted",
	})
}
