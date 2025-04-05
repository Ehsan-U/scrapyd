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
	"scrapyd/models"
	"scrapyd/services"
)

type GithubProject struct {
	Name   string `json:"name" binding:"required"`
	Url    string `json:"url" binding:"required"`
	Branch string `json:"branch"`
}

func ExtractSpidersFromSource(project *models.Project) error {
	absUrl := project.Url + "#" + project.Branch
	tempDir, relDockerfile, err := build.GetContextFromGitURL(absUrl, "Dockerfile")
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to get context from git url")
		return errs.ErrInvalidDockerfile
	}
	defer os.RemoveAll(tempDir)

	d, err := services.NewDaemon("localhost")
	if err != nil {
		return errs.ErrDaemonConnectionFailed
	}

	imageName := project.Name + ":latest"
	imageID, err := d.ImageIDByName(imageName)
	if err != nil {
		return errs.ErrDaemonNotResponding
	}
	if imageID != "" {
		if err = d.ImageRemove(imageID); err != nil {
			return errs.ErrDaemonNotResponding
		}
	}
	err = d.ImageBuild(tempDir, relDockerfile, project.Name)
	if err != nil {
		return errs.ErrInvalidDockerfile
	}

	spiders, err := d.SpiderList(project.Name)
	if err != nil {
		return errs.ErrSpidersNotFound
	}
	project.Spiders = spiders

	return nil
}

func ProjectCreate(c *gin.Context) {
	var gProject GithubProject
	var project models.Project

	if err := c.MustBindWith(&gProject, binding.JSON); err != nil {
		return
	}

	if !urlutil.IsGitURL(gProject.Url) {
		c.Error(errs.ErrInvalidGitUrl)
		return
	}

	// check DB
	if err := models.DB.First(&project, "Name = ?", gProject.Name).Error; err == nil {
		c.JSON(http.StatusCreated, types.Response{
			Status:  "success",
			Message: "exists",
		})
		return
	}

	project.Name = gProject.Name
	project.Url = gProject.Url
	project.Branch = gProject.Branch

	if err := ExtractSpidersFromSource(&project); err != nil {
		c.Error(err)
		return
	}

	models.DB.Create(&project)
	c.JSON(http.StatusCreated, types.Response{
		Status:  "success",
		Message: "created",
	})
}

func ProjectList(c *gin.Context) {
	var projects []models.Project

	models.DB.Find(&projects)
	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   projects,
	})
}

func ProjectGet(c *gin.Context) {
	var project models.Project

	id := c.Params.ByName("id")
	if err := models.DB.First(&project, id).Error; err != nil {
		c.Error(errs.ErrProjectNotFound)
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Status: "success",
		Data:   project,
	})
}

func ProjectUpdate(c *gin.Context) {
	var existingProject models.Project

	id := c.Params.ByName("id")
	if err := models.DB.First(&existingProject, id).Error; err != nil {
		c.Error(errs.ErrProjectNotFound)
		return
	}

	if err := ExtractSpidersFromSource(&existingProject); err != nil {
		c.Error(err)
		return
	}

	models.DB.Save(&existingProject)
	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "updated",
	})
}

func ProjectDelete(c *gin.Context) {
	var existingProject models.Project

	id := c.Params.ByName("id")
	if err := models.DB.First(&existingProject, id).Error; err != nil {
		c.Error(errs.ErrProjectNotFound)
		return
	}

	models.DB.Delete(&existingProject)
	c.JSON(http.StatusOK, types.Response{
		Status:  "success",
		Message: "deleted",
	})
}
