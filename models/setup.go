package models

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

func ConnectDatabase() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %s", err)
	}
	if err := db.AutoMigrate(&Server{}); err != nil {
		log.Fatalf("failed to auto migrate: %s", err)
	}
	DB = db
}
