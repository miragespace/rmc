package host

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	host := p.GetHost()
	if host == nil {
		return fmt.Errorf("Invalid heartbeat: nil Host")
	}
	name := host.GetName()
	if len(name) == 0 {
		return fmt.Errorf("Invalid heartbeat: empty host name")
	}
	now := time.Now()
	return m.db.Transaction(func(tx *gorm.DB) error {
		var existingHost Host
		lookupRes := tx.First(&existingHost, "name = ?", name)
		if errors.Is(lookupRes.Error, gorm.ErrRecordNotFound) {
			// register new host (TODO: capacity)
			existingHost = Host{
				Name:          name,
				LastHeartbeat: now,
			}
			createRes := tx.Create(&existingHost)
			return createRes.Error
		} else if lookupRes.Error == nil {
			// existing host: update timestamp (TODO: currently running instances)
			existingHost.LastHeartbeat = time.Now()
			saveRes := tx.Save(&existingHost)
			return saveRes.Error
		}
		return lookupRes.Error
	})
}
