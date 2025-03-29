package controllers

import (
	"fmt"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/builder/remotecontext/urlutil"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"log"
	"net/http"
	"os"
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
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "invalid github project url"})
		return
	}

	gProject.Url = gProject.Url + "#" + gProject.Branch
	tempDir, relDockerfile, err := build.GetContextFromGitURL(gProject.Url, "Dockerfile")
	if err != nil {
		err = fmt.Errorf("failed to get context from Git URL: %w", err)
		c.Error(err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	defer os.RemoveAll(tempDir)

	d, err := services.NewDaemon("localhost")
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	imageID := d.ImageIDByName(gProject.Name)
	if imageID != "" {
		d.ImageRemove(imageID)
	}
	err = d.ImageBuild(tempDir, relDockerfile, gProject.Name)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// get the spiders
	spiders, err := d.SpiderList(gProject.Name)
	if err != nil {
		c.Error(err)
	}
	log.Println(spiders)
	c.JSON(http.StatusCreated, gin.H{"message": "created"})
}
