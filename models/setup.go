package models

import (
	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	dsn := "my_database.db?_foreign_keys=on"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	//db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	if err := db.AutoMigrate(&Project{}, &Version{}, &Job{}); err != nil {
		log.Fatal().Err(err).Msg("failed to auto migrate")
	}
	DB = db
}
