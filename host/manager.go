package host

import (
	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Manager handles the database operations relating to Hosts
type Manager struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewManager returns a new Manager for hosts
func NewManager(logger *zap.Logger, db *gorm.DB) (*Manager, error) {
	if err := db.AutoMigrate(&Host{}); err != nil {
		return nil, extErrors.Wrap(err, "Cannot initilize host.Manager")
	}
	return &Manager{
		db:     db,
		logger: logger,
	}, nil
}

func (m *Manager) NextAvailableHost() (*Host, error) {
	panic("not implemented")
}

func (m *Manager) Heartbeat() error {
	panic("not implemented")
}
