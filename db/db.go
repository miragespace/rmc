package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func New(uri string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(uri), &gorm.Config{})
}
