package instance

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// Manager handles the database operations relating to Instance
type Manager struct {
	db *gorm.DB
	// TODO: add hooks to control server
	// TODO: add Subscription Client
}

// NewManager returns a new Manager for instances
func NewManager(db *gorm.DB) (*Manager, error) {
	if err := db.AutoMigrate(&Instance{}); err != nil {
		return nil, errors.Wrap(err, "Cannot initilize instance.Manager")
	}
	return &Manager{
		db: db,
	}, nil
}
