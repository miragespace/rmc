package instance

import (
	"context"
	"database/sql"
	"errors"

	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Manager handles the database operations relating to Instance
type Manager struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewManager returns a new Manager for instances
func NewManager(logger *zap.Logger, db *gorm.DB) (*Manager, error) {
	if err := db.AutoMigrate(&Instance{}); err != nil {
		return nil, extErrors.Wrap(err, "Cannot initilize instance.Manager")
	}
	return &Manager{
		db:     db,
		logger: logger,
	}, nil
}

// Create will insert an Instance record to the database
func (m *Manager) Create(ctx context.Context, inst *Instance) error {
	result := m.db.WithContext(ctx).Create(inst)
	if result.Error != nil {
		m.logger.Error("Unable to create new instance in database",
			zap.Error(result.Error),
		)
		return extErrors.Wrap(result.Error, "Cannot create instance")
	}
	return nil
}

// GetByID will attempt to lookup and return an Instance record by Instance ID
func (m *Manager) GetByID(ctx context.Context, id string) (*Instance, error) {
	inst := Instance{}

	result := m.db.WithContext(ctx).Where("id = ?", id).First(&inst)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		m.logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, extErrors.Wrap(result.Error, "Cannot get instance by id")
	}

	return &inst, nil
}

// GetBySubscriptionID will attempt to lookup and return an Instance record by Subscription ID
func (m *Manager) GetBySubscriptionID(ctx context.Context, sid string) (*Instance, error) {
	inst := Instance{}

	result := m.db.WithContext(ctx).Where("subscription_id = ?", sid).First(&inst)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		m.logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, extErrors.Wrap(result.Error, "Cannot get instance by subscription id")
	}

	return &inst, nil
}

// Update will update an Instance record without transaction. If transaction is required, use LambdaUpdate
func (m *Manager) Update(ctx context.Context, inst *Instance) error {
	result := m.db.WithContext(ctx).Save(inst)

	if result.Error != nil {
		m.logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return result.Error
	}
	return nil
}

// List will return all Instance records by Customer ID. If all == true, it will also return Status == Terminated
func (m *Manager) List(ctx context.Context, cid string, all bool) ([]Instance, error) {
	results := make([]Instance, 0, 1)
	baseQuery := m.db.WithContext(ctx).Order("created_at desc")
	if !all {
		baseQuery = baseQuery.Where("status = ?", StatusActive)
	}
	result := baseQuery.Find(&results, "customer_id = ?", cid)
	if result.Error != nil {
		m.logger.Error("Database returned error",
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
	var result LambdaResult
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current Instance
		lookupRes := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&current, "id = ?", id)
		if lookupRes.Error == nil {
			var desired Instance = current
			shouldSave, returnValue := lambda(&current, &desired)
			if shouldSave {
				if saveRes := tx.Save(&desired); saveRes.Error != nil {
					return saveRes.Error
				}
			}
			result.Instance = &desired
			result.ReturnValue = returnValue
			return nil
		} else if errors.Is(lookupRes.Error, gorm.ErrRecordNotFound) {
			_, returnValue := lambda(nil, nil)
			result.Instance = nil
			result.ReturnValue = returnValue
			return nil
		}
		return lookupRes.Error
	}, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})

	result.TxError = err
	return result
}
