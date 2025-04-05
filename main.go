package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"scrapyd/api/errs"
	"scrapyd/api/types"
	"scrapyd/controllers"
	"scrapyd/models"
	"time"
)

func ZLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		latency := time.Since(startTime)
		status := c.Writer.Status()

		if len(c.Errors) != 0 {
			err := c.Errors.Last().Err
			log.Error().Err(err).Msg("")

			if !c.Writer.Written() {
				for knownErr, statusCode := range errs.ErrStatusMap {
					if errors.Is(err, knownErr) {
						c.AbortWithStatusJSON(statusCode, types.Response{
							Status:  "error",
							Message: knownErr.Error(),
						})
						break
					}
				}
			}
		}

		log.Debug().
			Int("status", status).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Msg("")
	}
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	//gin.DefaultWriter = zerolog.New(os.Stdout).With().Timestamp().Logger()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	gin.DefaultWriter = zerolog.ConsoleWriter{Out: os.Stdout}
	//gin.SetMode("release")
	router := gin.New()
	router.Use(ZLogMiddleware(), gin.Recovery())

	models.ConnectDatabase()

	// server CRUD
	router.POST("/servers", controllers.ServerCreate)
	router.GET("/servers", controllers.ServerList)
	router.GET("/servers/:id", controllers.ServerGet)
	router.PATCH("/servers/:id", controllers.ServerUpdate)
	router.DELETE("/servers/:id", controllers.ServerDelete)

	// project CRUD
	router.POST("/projects", controllers.ProjectCreate)
	router.GET("/projects", controllers.ProjectList)
	router.GET("/projects/:id", controllers.ProjectGet)
	router.PATCH("/projects/:id", controllers.ProjectUpdate)
	router.DELETE("/projects/:id", controllers.ProjectDelete)

	// deployments CRUD
	router.POST("/deployments", controllers.DeploymentCreate)
	router.GET("/deployments", controllers.DeploymentList)
	router.GET("/deployments/:id", controllers.DeploymentGet)
	//router.PATCH("/deployments/:id", controllers.DeploymentUpdate)
	router.DELETE("/deployments/:id", controllers.DeploymentDelete)

	if err := router.Run(":8080"); err != nil {
		log.Fatal().Err(err).Msg("app failed to start")
	}
}
