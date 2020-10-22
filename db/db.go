package db

import (
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// New returns an instance for interacting with the PostgreSQL database
func New(uri string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(uri), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "Cannot connect to database")
	}
	pool, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get the connection pool")
	}
	pool.SetMaxIdleConns(1)
	pool.SetMaxOpenConns(20)
	pool.SetConnMaxLifetime(time.Hour)
	return db, nil
}
