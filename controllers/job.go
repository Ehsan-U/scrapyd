package controllers

import (
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"io"
	"net/http"
	"scrapyd/api/errs"
	"scrapyd/api/types"
	"scrapyd/models"
	"scrapyd/services"
	"scrapyd/tasks"
	"slices"
	"strings"
)

type flushingWriter struct {
	writer  io.Writer
	flusher http.Flusher
}

func (fw *flushingWriter) Write(p []byte) (int, error) {
	n, err := fw.writer.Write(p)
	if err == nil {
		fw.flusher.Flush()
	}
	return n, err
}

func JobCreate(c *gin.Context) {
	var request types.JobRequest

	if err := c.MustBindWith(&request, binding.JSON); err != nil {
		return
	}
	if err := models.DB.First(&models.Project{}, "id = ?", request.ProjectID).Error; err != nil {
		c.Error(errs.ErrProjectNotFound)
		return
	}
	var version models.Version
	if err := models.DB.First(&version, "id = ?", request.VersionID).Error; err != nil {
		c.Error(errs.ErrVersionNotFound)
		return
	}
	if !slices.Contains(version.Spiders, request.Spider) {
		c.Error(errs.ErrSpiderNotFound)
		return
	}
	if request.ID != "" {
		if err := models.DB.First(&models.Job{}, "id = ?", request.ID).Error; err == nil {
			c.Error(errs.ErrJobConflict)
			return
		}
	}

	if request.ID == "" {
		reqID, _ := uuid.NewUUID()
		request.ID = strings.ReplaceAll(reqID.String(), "-", "")
	}
	job := models.Job{
		ID:        request.ID,
		ProjectID: request.ProjectID,
		VersionID: request.VersionID,
		Status:    "pending",
		Spider:    request.Spider,
		Setting:   request.Setting,
	}

	if err := tasks.NewTask("execute:job", job.ID); err != nil {
		c.Error(err)
		return
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
	if err := models.DB.Preload("Project").Preload("Version").First(&job, "id = ?", id).Error; err != nil {
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
		ID     string `json:"id" binding:"required"`
		Status string `json:"status" binding:"required"`
	}

	if err := c.MustBindWith(&updateData, binding.JSON); err != nil {
		return
	}

	if err := models.DB.First(&existingJob, "id = ?", updateData.ID).Error; err != nil {
		c.Error(errs.ErrJobNotFound)
		return
	}

	if updateData.Status == "cancel" {
		if err := tasks.NewTask("cancel:job", updateData.ID); err != nil {
			c.Error(err)
			return
		}
	}
	if updateData.Status == "restart" {
		if err := tasks.NewTask("restart:job", updateData.ID); err != nil {
			c.Error(err)
			return
		}
	}

	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "updated",
	})
}

func JobDelete(c *gin.Context) {
	var job models.Job

	id := c.Params.ByName("id")
	if err := models.DB.First(&job, "id = ?", id).Error; err != nil {
		c.Error(errs.ErrJobNotFound)
		return
	}

	// cleanup the related stuff like container
	if err := services.JobCleanup(&job); err != nil {
		c.Error(err)
		return
	}

	models.DB.Delete(&job)
	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "deleted",
	})
}

func JobLogStream(c *gin.Context) {
	var job models.Job

	id := c.Params.ByName("id")
	if err := models.DB.First(&job, "id = ?", id).Error; err != nil {
		c.Error(errs.ErrJobNotFound)
		return
	}

	reqCtx := c.Request.Context()
	reader, err := services.JobLogReader(reqCtx, &job)
	if err != nil {
		c.Error(err)
		return
	}
	defer reader.Close()

	flusher, _ := c.Writer.(http.Flusher)
	fw := &flushingWriter{writer: c.Writer, flusher: flusher}

	stdcopy.StdCopy(fw, fw, reader)
}
