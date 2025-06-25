package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/signal"
	"scrapyd/api/errs"
	"scrapyd/api/types"
	"scrapyd/controllers"
	"scrapyd/listerners"
	"scrapyd/models"
	"sync"
	"syscall"
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
			// catch rogue err
			if !c.Writer.Written() {
				c.AbortWithStatusJSON(http.StatusInternalServerError, types.Response{
					Status:  "error",
					Message: "",
				})
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
	var wg sync.WaitGroup
	models.ConnectDatabase()

	mainCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		listerners.StartDockerEventListener(mainCtx)
	}()

	router := gin.New()
	router.Use(ZLogMiddleware(), gin.Recovery())
	srv := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}
	// Project
	router.POST("/projects", controllers.ProjectCreate)       // AddVersion
	router.GET("/projects", controllers.ProjectList)          // ListProjects
	router.DELETE("/projects/:id", controllers.ProjectDelete) // DelProject

	// Version
	router.POST("/versions", controllers.VersionCreate)                           // AddVersion
	router.GET("/versions/:project_id", controllers.VersionList)                  // ListVersions ListSpiders
	router.DELETE("/versions/:project_id/:version_id", controllers.VersionDelete) // DelVersion

	// Jobs
	router.POST("/jobs", controllers.JobCreate)  // Schedule
	router.GET("/jobs", controllers.JobList)     // ListJobs
	router.GET("/jobs/:id", controllers.JobGet)  // Status
	router.PATCH("/jobs", controllers.JobUpdate) // Cancel
	router.DELETE("/jobs/:id", controllers.JobDelete)
	router.GET("/jobs/:id/logs", controllers.JobLogStream)

	// miscellaneous
	router.GET("/daemonstatus", controllers.DaemonStatus) // DaemonStatus

	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.ListenAndServe()
	}()

	<-mainCtx.Done()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server graceful shutdown failed")
	}

	wg.Wait()
}
