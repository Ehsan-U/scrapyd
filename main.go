package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"scrapyd/controllers"
	"scrapyd/models"
	"time"
)

type Response struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

func ZLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		latency := time.Since(startTime)
		status := c.Writer.Status()
		if len(c.Errors) != 0 {
			err := c.Errors.Last()
			log.Error().
				Int("status", status).
				Err(err).
				Msg("")
		} else {
			log.Debug().
				Int("status", status).
				Dur("latency", latency).
				Str("ip", c.ClientIP()).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Msg("")
		}
	}
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	gin.DefaultWriter = zerolog.New(os.Stdout).With().Timestamp().Logger()
	gin.SetMode("release")
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

	if err := router.Run(":8080"); err != nil {
		log.Fatal().Err(err).Msg("app failed to start")
	}
}
