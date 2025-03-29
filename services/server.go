package services

import (
	"log"
	"scrapyd/models"
)

func SetServerStatus(server *models.Server, status string) {
	models.DB.Model(server).
		Where("Id = ?", server.Id).
		Update("status", status)
	log.Printf("server [%s] status changed: %s -> %s", server.Id, server.Status, status)
}
