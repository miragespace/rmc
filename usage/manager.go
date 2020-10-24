package usage

import (
	"context"
	"errors"
	"fmt"
	"time"

	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Manager struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewManager(logger *zap.Logger, db *gorm.DB) (*Manager, error) {
	return &Manager{
		db:     db,
		logger: logger,
	}, nil
}

func (m *Manager) RecordUsage(ctx context.Context, sid string, event Event) error {
	result := m.db.Transaction(func(tx *gorm.DB) error {
		var usage Usage
		lookupRes := tx.Order("start desc").Where("subscription_id = ?", sid).First(&usage)
		if event == Start {
			if usage.End == nil {
				return fmt.Errorf("Cannot record usage for START if it has not END'd")
			}
			if errors.Is(lookupRes.Error, gorm.ErrRecordNotFound) {
				createRes := tx.Create(&Usage{
					SubscriptionID: sid,
					Start:          time.Now(),
				})
				return createRes.Error
			}
			return lookupRes.Error
		}
		// event == End
		if errors.Is(lookupRes.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("Cannot record usage for END it has not START'd")
		}
		now := time.Now()
		usage.End = &now
		updateRes := tx.Save(&usage)
		return updateRes.Error
	})
	if result != nil {
		return extErrors.Wrap(result, "Cannot record usage")
	}
	return nil
}
