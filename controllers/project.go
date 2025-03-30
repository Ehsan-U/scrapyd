package controllers

import (
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/builder/remotecontext/urlutil"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"scrapyd/api/errs"
	"scrapyd/api/types"
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
		c.Error(errs.ErrInvalidGitUrl)
		return
	}

	gProject.Url = gProject.Url + "#" + gProject.Branch
	tempDir, relDockerfile, err := build.GetContextFromGitURL(gProject.Url, "Dockerfile")
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to get context from git url")
		c.Error(errs.ErrInvalidDockerfile)
		return
	}
	defer os.RemoveAll(tempDir)

	d, err := services.NewDaemon("localhost")
	if err != nil {
		c.Error(errs.ErrDaemonConnectionFailed)
		return
	}

	imageName := gProject.Name + ":latest"
	imageID, err := d.ImageIDByName(imageName)
	if err != nil {
		c.Error(errs.ErrDaemonNotResponding)
		return
	}
	if imageID != "" {
		if err = d.ImageRemove(imageID); err != nil {
			c.Error(errs.ErrDaemonNotResponding)
			return
		}
	}
	err = d.ImageBuild(tempDir, relDockerfile, gProject.Name)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, types.Response{
			Status:  "error",
			Data:    err.Error(),
			Message: errs.ErrInvalidDockerfile.Error(),
		})
		return
	}

	_, err = d.SpiderList(gProject.Name)
	if err != nil {
		c.Error(errs.ErrSpidersNotFound)
		return
	}

	c.JSON(http.StatusCreated, types.Response{
		Status:  "success",
		Message: "created",
	})
}
