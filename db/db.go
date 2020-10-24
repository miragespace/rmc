package db

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"moul.io/zapgorm2"
)

type patchedLogger struct {
	zapgorm2.Logger
}

// ErrRecordNotFound will be handled in application logic, let's not forward this to zap/sentry
func (l *patchedLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if err == gorm.ErrRecordNotFound {
		return
	}
	l.Logger.Trace(ctx, begin, fc, err)
}

// New returns an instance for interacting with the PostgreSQL database
func New(logger *zap.Logger, uri string) (*gorm.DB, error) {
	gLogger := zapgorm2.Logger{
		ZapLogger:        logger,
		LogLevel:         gormlogger.Warn,
		SlowThreshold:    time.Second,
		SkipCallerLookup: false,
	}
	db, err := gorm.Open(postgres.Open(uri), &gorm.Config{
		Logger: &patchedLogger{
			Logger: gLogger,
		},
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
