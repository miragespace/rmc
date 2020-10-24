package db

import (
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"moul.io/zapgorm2"
)

// New returns an instance for interacting with the PostgreSQL database
func New(logger *zap.Logger, uri string) (*gorm.DB, error) {
	gLogger := &zapgorm2.Logger{
		ZapLogger:        logger,
		LogLevel:         gormlogger.Warn,
		SlowThreshold:    time.Second,
		SkipCallerLookup: false,
	}
	db, err := gorm.Open(postgres.Open(uri), &gorm.Config{
		Logger: gLogger,
	})
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
