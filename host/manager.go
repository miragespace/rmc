package host

import (
	"context"

	"github.com/zllovesuki/rmc/spec/protocol"

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

func (m *Manager) GetHostByName(ctx context.Context, name string) (*Host, error) {
	panic("not implemented")
}

func (m *Manager) NextAvailableHost(ctx context.Context) (*Host, error) {
	panic("not implemented")
}

func (m *Manager) ProcessHeartbeat(ctx context.Context, p *protocol.Heartbeat) error {
	panic("not implemented")
}
