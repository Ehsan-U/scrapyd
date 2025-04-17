package controllers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"scrapyd/api/errs"
	"scrapyd/api/types"
	"scrapyd/models"
	"scrapyd/services"
	"scrapyd/tasks"
	"slices"
	"strings"
	"time"
)

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

	d, err := services.NewDaemon()
	if err != nil {
		c.Error(err)
		return
	}

	contName := fmt.Sprintf("%s_%s_%s_%s", job.ID, job.ProjectID, job.VersionID, job.Spider)
	cont, err := d.FindContainerByName(contName)
	if err != nil {
		c.Error(err)
		return
	}

	reqCtx := c.Request.Context()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	reader, err := d.ContainerLogs(ctx, cont.ID)
	if err != nil {
		c.Error(err)
		return
	}
	defer reader.Close()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		log.Debug().Msg("writer does not support flushing")
		return
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		select {
		case <-reqCtx.Done():
			log.Debug().Msgf("Client disconnected for job %s logs. Stopping stream.", job.ID)
			return
		default:
			_, err := fmt.Fprintf(c.Writer, "%s", scanner.Text())
			if err != nil {
				log.Error().Err(err).Msgf("Error writing to client for job %s", job.ID)
				return
			}
			flusher.Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		if !(errors.Is(err, context.Canceled) || errors.Is(err, io.EOF)) {
			log.Printf("Error reading log stream for job %s: %v", job.ID, err)
		} else if errors.Is(err, io.EOF) {
			fmt.Fprintf(c.Writer, "event: stream_end\ndata: Log stream ended.\n\n")
			flusher.Flush()
		}
	}
	log.Printf("Finished streaming logs for job: %s", job.ID)
}
