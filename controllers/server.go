package controllers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"time"

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

	server := models.Server{
		Name:    host.Name,
		Address: host.Address,
	}
	dClient, err := services.NewDaemon(&server)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	systemInfo, err := dClient.GetSystemInfo(ctx)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	server.Id = systemInfo.ID
	server.Status = "up"
	server.HostName = systemInfo.Name
	server.CPU = systemInfo.NCPU
	server.Memory = systemInfo.MemTotal

	result := models.DB.FirstOrCreate(&server)
	if result.RowsAffected == 1 {
		c.JSON(http.StatusCreated, gin.H{"message": "created"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "exists"})
	}
}

func ServerList(c *gin.Context) {
	var servers []models.Server

	models.DB.Find(&servers)
	c.JSON(http.StatusOK, gin.H{"message": servers})
}

func ServerUpdate(c *gin.Context) {
	var existingServer models.Server

	id := c.Params.ByName("id")
	err := models.DB.First(&existingServer, "Id = ?", id).Error
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
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
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func ServerGet(c *gin.Context) {
	var existingServer models.Server

	id := c.Params.ByName("id")
	err := models.DB.First(&existingServer, "Id = ?", id).Error
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": existingServer})
}

func ServerDelete(c *gin.Context) {
	var existingServer models.Server

	id := c.Params.ByName("id")
	err := models.DB.First(&existingServer, "Id = ?", id).Error
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	models.DB.Delete(&existingServer)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	return
}
