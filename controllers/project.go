package controllers

import (
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/builder/remotecontext/urlutil"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"scrapyd/api"
	"scrapyd/services"
)

type GithubProject struct {
	Name   string `json:"name" binding:"required"`
	Url    string `json:"url" binding:"required"`
	Branch string `json:"branch"`
}

func ProjectCreate(c *gin.Context) {
	var gProject GithubProject

	if err := c.MustBindWith(&gProject, binding.JSON); err != nil {
		return
	}

	if !urlutil.IsGitURL(gProject.Url) {
		c.JSON(http.StatusUnprocessableEntity, api.Response{
			Status:  "error",
			Message: "please provide a valid git url",
		})
		return
	}

	gProject.Url = gProject.Url + "#" + gProject.Branch
	tempDir, relDockerfile, err := build.GetContextFromGitURL(gProject.Url, "Dockerfile")
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to get context from git url")
		c.JSON(http.StatusNotFound, api.Response{
			Status:  "error",
			Message: "please ensure git project contains a dockerfile",
		})
		return
	}
	defer os.RemoveAll(tempDir)

	d, err := services.NewDaemon("localhost")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, api.Response{
			Status:  "error",
			Message: "docker daemon not available on localhost",
		})
		return
	}
	imageID := d.ImageIDByName(gProject.Name)
	if imageID != "" {
		d.ImageRemove(imageID)
	}
	err = d.ImageBuild(tempDir, relDockerfile, gProject.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Status:  "error",
			Message: "invalid dockerfile",
		})
		return
	}

	// get the spiders
	_, err = d.SpiderList(gProject.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Status:  "error",
			Message: "no spiders found in scrapy project",
		})
	}
	c.JSON(http.StatusCreated, api.Response{
		Status:  "success",
		Message: "created",
	})
}
