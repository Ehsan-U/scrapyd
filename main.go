package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"scrapyd/controllers"
	"scrapyd/models"
)

func main() {
	//gin.SetMode("release")
	router := gin.Default()
	models.ConnectDatabase()

	// server CRUD
	router.POST("/servers", controllers.ServerCreate)
	router.GET("/servers", controllers.ServerList)
	router.GET("/servers/:id", controllers.ServerGet)
	router.PATCH("/servers/:id", controllers.ServerUpdate)
	router.DELETE("/servers/:id", controllers.ServerDelete)

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("server failed to start: %s", err)
	}
}
