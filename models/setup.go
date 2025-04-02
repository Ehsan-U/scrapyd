package models

import (
	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	if err := db.AutoMigrate(&Server{}, &Project{}); err != nil {
		log.Fatal().Err(err).Msg("failed to auto migrate")
	}
	DB = db
}
