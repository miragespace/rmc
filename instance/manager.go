package instance

import (
	"context"
	"errors"

	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

func (m *Manager) NewInstance(ctx context.Context, inst *Instance) error {
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

func (m *Manager) UpdateInstance(ctx context.Context, inst *Instance) error {
	result := m.db.WithContext(ctx).Save(inst)

	if result.Error != nil {
		m.logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return result.Error
	}
	return nil
}
