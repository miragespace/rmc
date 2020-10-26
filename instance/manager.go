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

func (m *Manager) GetBySubscriptionId(ctx context.Context, sid string) (*Instance, error) {
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

// LambdaUpdateFunc is used when transaction is required for update. Return value determines if InstanceManager should commit the changes.
// Note that current and desired may be nil if no Instance with given id was found, and must return false if that is the case
type LambdaUpdateFunc func(current *Instance, desired *Instance) (shouldSave bool)

// LambdaUpdate will perform a transactional update based on the lambda function. If the lambda signals shouldSave AND update was successful, it will return the new state.
// The selected Instance will be locked with FOR UPDATE
func (m *Manager) LambdaUpdate(ctx context.Context, id string, lambda LambdaUpdateFunc) (*Instance, error) {
	var desired Instance
	var shouldReturn bool
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current Instance
		lookupRes := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&current, "id = ?", id)
		if lookupRes.Error == nil {
			desired = current
			if lambda(&current, &desired) {
				if saveRes := tx.Save(&desired); saveRes.Error != nil {
					return saveRes.Error
				}
				shouldReturn = true
			}
			return nil
		} else if errors.Is(lookupRes.Error, gorm.ErrRecordNotFound) {
			lambda(nil, nil)
			return nil
		}
		return lookupRes.Error
	}, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		// transaction failed, return nil new state
		return nil, err
	}
	if !shouldReturn {
		// shouldSave == false, return nil new state
		return nil, nil
	}
	// transaction succeed and shouldSave == true, return new state
	return &desired, nil
}
