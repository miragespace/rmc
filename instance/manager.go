package instance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zllovesuki/rmc/subscription"

	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ManagerOptions struct {
	DB                  *gorm.DB
	Logger              *zap.Logger
	SubscriptionManager *subscription.Manager // TODO: use event sourcing instead of requiring SubscriptionManager as dependency
}

// Manager handles the database operations relating to Instance
type Manager struct {
	ManagerOptions
}

func historyWithLimit(limit int) func(*gorm.DB) *gorm.DB {
	return func(s *gorm.DB) *gorm.DB {
		baseQuery := s.Order("histories.timestamp desc")
		if limit > 0 {
			baseQuery = baseQuery.Limit(limit)
		}
		return baseQuery
	}
}

// NewManager returns a new Manager for instances
func NewManager(option ManagerOptions) (*Manager, error) {
	if option.DB == nil {
		return nil, fmt.Errorf("nil DB is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	if option.SubscriptionManager == nil {
		return nil, fmt.Errorf("SubscriptionManager is required for usage reporting")
	}
	if err := option.DB.AutoMigrate(&Instance{}, &History{}); err != nil {
		return nil, extErrors.Wrap(err, "Cannot initilize instance.Manager")
	}
	return &Manager{
		ManagerOptions: option,
	}, nil
}

// Create will insert an Instance record to the database
func (m *Manager) Create(ctx context.Context, inst *Instance) error {
	result := m.DB.WithContext(ctx).Create(inst)
	if result.Error != nil {
		m.Logger.Error("Unable to create new instance in database",
			zap.Error(result.Error),
		)
		return extErrors.Wrap(result.Error, "Cannot create instance")
	}
	return nil
}

// GetOption is used when querying for a single Instance record
type GetOption struct {
	InstanceID     string
	SubscriptionID string
	WithHistory    bool
}

// Get will attempt to lookup and return an Instance record as specified in GetOption
func (m *Manager) Get(ctx context.Context, opt GetOption) (*Instance, error) {
	baseQuery := m.DB.WithContext(ctx)
	if len(opt.InstanceID) > 0 {
		baseQuery = baseQuery.Where("id = ?", opt.InstanceID)
	} else if len(opt.SubscriptionID) > 0 {
		baseQuery = baseQuery.Where("subscription_id = ?", opt.SubscriptionID)
	} else {
		return nil, fmt.Errorf("Either GetOption.InstanceID or GetOption.SubscriptionID is required")
	}
	if opt.WithHistory {
		baseQuery = baseQuery.Preload("Histories", historyWithLimit(5))
	}
	inst := Instance{}
	result := baseQuery.First(&inst)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		m.Logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, extErrors.Wrap(result.Error, "Cannot get instance by id")
	}

	return &inst, nil
}

// ListOption is used when querying for a list of Instance records
type ListOption struct {
	CustomerID        string
	SubscriptionID    string
	IncludeTerminated bool
	Before            time.Time
	Limit             int
}

// List will return all Instance records as specified in ListOption
func (m *Manager) List(ctx context.Context, opt ListOption) ([]Instance, error) {
	baseQuery := m.DB.WithContext(ctx).Order("created_at desc")
	if len(opt.CustomerID) > 0 {
		baseQuery = baseQuery.Where("customer_id = ?", opt.CustomerID)
	} else if len(opt.SubscriptionID) > 0 {
		baseQuery = baseQuery.Where("subscription_id = ?", opt.SubscriptionID)
	} else {
		return nil, fmt.Errorf("Either ListOption.CustomerID or ListOption.SubscriptionID is required")
	}
	if !opt.IncludeTerminated {
		baseQuery = baseQuery.Where("status = ?", StatusActive)
	}
	if opt.Limit > 0 {
		baseQuery = baseQuery.Limit(opt.Limit)
	}
	if !opt.Before.IsZero() {
		baseQuery = baseQuery.Where("created_at < ?", opt.Before)
	}

	results := make([]Instance, 0, 1)
	result := baseQuery.Find(&results)

	if result.Error != nil {
		m.Logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, result.Error
	}
	return results, nil
}

// LambdaUpdateFunc is used when transaction is required for update. shouldSave determines if InstanceManager should commit the changes.
// Note that current and desired may be nil if no Instance with given id was found, and must return false if that is the case.
// returnValue is for the function to return useful information after the lambda has been executed. Caller should check the value of LambdaResult.ReturnValue
// for additional context and handling.
// Note that the transaction may be retried multiple times, therefore the lambda function should not introduce side effects.
type LambdaUpdateFunc func(current *Instance, desired *Instance) (shouldSave bool, returnValue interface{})

// LambdaResult contains the result of lambda execution. Instance will only be populated if lambda signals shouldSave AND update was successful,
// LambdaResult.ReturnValue will contain the return value from the lambda function
type LambdaResult struct {
	Instance    *Instance
	ReturnValue interface{}
	TxError     error
}

// LambdaUpdate will perform a transactional update based on the lambda function.
// The selected Instance will be locked with FOR UPDATE
func (m *Manager) LambdaUpdate(ctx context.Context, id string, lambda LambdaUpdateFunc) LambdaResult {
	logger := m.Logger.With(
		zap.String("InstanceID", id),
	)

	var result LambdaResult
	err := m.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current Instance
		lookupRes := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Histories", historyWithLimit(2)).
			First(&current, "id = ?", id)
		if lookupRes.Error == nil {
			var desired Instance = current
			shouldSave, returnValue := lambda(&current, &desired)
			if shouldSave {
				if saveRes := tx.Save(&desired); saveRes.Error != nil {
					logger.Error("Cannot save Instance changes",
						zap.Error(saveRes.Error),
					)
					return saveRes.Error
				}
			}
			result.Instance = &desired
			result.ReturnValue = returnValue
			return m.recordHistory(ctx, tx, &desired)
		} else if errors.Is(lookupRes.Error, gorm.ErrRecordNotFound) {
			_, returnValue := lambda(nil, nil)
			result.Instance = nil
			result.ReturnValue = returnValue
			return nil
		}
		logger.Error("Cannot lookup Instance by ID",
			zap.Error(lookupRes.Error),
		)
		return lookupRes.Error
	}, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})

	result.TxError = err
	return result
}

func (m *Manager) recordHistory(ctx context.Context, tx *gorm.DB, instance *Instance) error {
	logger := m.Logger.With(
		zap.String("InstanceID", instance.ID),
	)

	now := time.Now()

	// insert a History record here
	if instance.PreviousState != instance.State {
		if histRes := tx.Create(&History{
			InstanceID: instance.ID,
			Timestamp:  now,
			State:      instance.State,
		}); histRes.Error != nil {
			logger.Error("Cannot insert History log",
				zap.Error(histRes.Error),
			)
			return histRes.Error
		}
	}

	if instance.PreviousState != StateStopping || instance.State != StateStopped {
		return nil
	}

	// If the State becomes "StateStopped", look back to find the last
	// timestamp when it was "StateRunning". Then SubscriptionManager
	// should be able to determine how much usage we need to record (if any)
	var previousLog History
	pastRes := tx.Order("histories.timestamp desc").
		Limit(1).
		Where("instance_id = ? AND timestamp < ? AND state IN ?",
			instance.ID,
			now,
			[]string{StateRunning, StateStopped},
		).
		First(&previousLog)

	// ErrRecordNotFound is an error, and if current State is "Stopped",
	// there must exists a corresponding "Start" history
	if pastRes.Error != nil {
		logger.Error("Cannot look back History logs to determine usage",
			zap.Error(pastRes.Error),
		)
		return pastRes.Error
	}
	if previousLog.State != StateRunning {
		logger.Error("Unable to find corresponding previous Start history log",
			zap.Error(pastRes.Error),
		)
		return fmt.Errorf("Previous History log is inconsistent with current State (expected: Running, actual: %s", previousLog.State)
	}

	opt := subscription.RecordUsageOption{
		CustomerID:     instance.CustomerID,
		SubscriptionID: instance.SubscriptionID,

		CurrentlyRunning:  false,
		PreviousStartDate: &previousLog.Timestamp,
		CurrentEndDate:    &now,
		Primary:           true,
	}
	if err := m.SubscriptionManager.RecordUsage(ctx, opt); err != nil {
		logger.Error("Unable to record usage in SubscriptionManager",
			zap.Error(err),
		)
		return err
	}

	return nil
}
