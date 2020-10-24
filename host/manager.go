package host

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/protocol"

	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// good god there has to be a better way for this
var interval = (spec.HeartbeatInterval * 2).String()
var nextHostQuery string = "? - last_heartbeat < interval '" + interval[:len(interval)-1] + " seconds' AND running + stopped < capacity"

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

// GetHostByName looks up a Host by its registered name
func (m *Manager) GetHostByName(ctx context.Context, name string) (*Host, error) {
	var host Host
	result := m.db.WithContext(ctx).First(&host, "name = ?", name)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		m.logger.Error("Unable to get host by name",
			zap.Error(result.Error),
		)
		return nil, extErrors.Wrap(result.Error, "Cannot get host by name")
	}
	return &host, nil
}

// NextAvailableHost looks up an available host for provisioning. If it can't find one, it will be nil
func (m *Manager) NextAvailableHost(ctx context.Context) (*Host, error) {
	/*
		Criteria for an "available" host:
		1. Last heartbeart was in the last (2 * HeartbeatInterval) seconds
		2. Has capacity > 0
		3. Has (running + stopped) < capacity (implied by 2)
	*/
	hosts := make([]Host, 1)
	result := m.db.WithContext(ctx).
		Order("random()").
		Limit(1).
		Where(
			nextHostQuery,
			time.Now(),
		).Find(&hosts)

	if result.Error != nil {
		m.logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, extErrors.Wrap(result.Error, "Cannot get next available host")
	}

	if len(hosts) == 0 {
		return nil, nil
	}

	return &hosts[0], nil
}

// ProcessHeartbeat will process the heartbeats from hosts and update their status
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
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingHost Host
		lookupRes := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&existingHost, "name = ?", name)
		if errors.Is(lookupRes.Error, gorm.ErrRecordNotFound) {
			// register new host
			existingHost = Host{
				Name:          name,
				Running:       0,
				Stopped:       0,
				Capacity:      host.GetCapacity(),
				LastHeartbeat: now,
			}
			createRes := tx.Create(&existingHost)
			return createRes.Error
		} else if lookupRes.Error == nil {
			// existing host
			existingHost.LastHeartbeat = time.Now()
			existingHost.Running = host.GetRunning()
			existingHost.Stopped = host.GetStopped()
			existingHost.Capacity = host.GetCapacity()
			saveRes := tx.Save(&existingHost)
			return saveRes.Error
		}
		return lookupRes.Error
	})
}
