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

func ServerCreate(c *gin.Context) {
	var request types.ServerRequest
	var server models.Server

	if err := c.MustBindWith(&request, binding.JSON); err != nil {
		return
	}

	if err := models.DB.First(&server, "name = ?", request.Name).Error; err == nil {
		c.Error(errs.ErrServerConflict)
		return
	}

	d, err := services.NewDaemon(request.Address)
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

	server.Name = request.Name
	server.Address = request.Address
	server.Status = "up"
	server.HostName = systemInfo.Name

	models.DB.Create(&server)
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

func ServerGet(c *gin.Context) {
	var server models.Server

	id := c.Params.ByName("id")
	if err := models.DB.First(&server, id).Error; err != nil {
		c.Error(errs.ErrServerNotFound)
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   server,
	})
}

func ServerDelete(c *gin.Context) {
	var server models.Server

	id := c.Params.ByName("id")
	if rows := models.DB.Delete(&server, id).RowsAffected; rows == 0 {
		c.Error(errs.ErrProjectNotFound)
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "deleted",
	})
}
