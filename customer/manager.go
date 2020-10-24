package customer

import (
	"context"
	"errors"

	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/client"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Manager handles the database operations relating to Customers
type Manager struct {
	stripeClient *client.API
	db           *gorm.DB
	logger       *zap.Logger
}

// NewManager returns a new Manager for customers
func NewManager(logger *zap.Logger, db *gorm.DB, s *client.API) (*Manager, error) {
	if err := db.AutoMigrate(&Customer{}); err != nil {
		return nil, extErrors.Wrap(err, "Cannot initilize customer.Manager")
	}
	return &Manager{
		stripeClient: s,
		db:           db,
		logger:       logger,
	}, nil
}

// New will create a new customer profile in Stripe and in the database
func (m *Manager) New(ctx context.Context, email string) (*Customer, error) {
	listParams := &stripe.CustomerListParams{
		ListParams: stripe.ListParams{
			Context: ctx,
		},
		Email: stripe.String(email),
	}
	listParams.Filters.AddFilter("limit", "", "1")
	l := m.stripeClient.Customers.List(listParams)

	if l.Err() != nil {
		m.logger.Error("Stripe returned error",
			zap.Error(l.Err()),
		)
		return nil, extErrors.Wrap(l.Err(), "Cannot query customer by email to Stripe")
	}

	var c *stripe.Customer
	for l.Next() {
		c = l.Customer()
	}

	if c == nil {
		// actually a new customer
		var err error
		params := &stripe.CustomerParams{
			Params: stripe.Params{
				Context: ctx,
			},
			Email: stripe.String(email),
		}
		c, err = m.stripeClient.Customers.New(params)
		if err != nil {
			m.logger.Error("Stripe returned error",
				zap.Error(err),
			)
			return nil, extErrors.Wrap(err, "Cannot create a new Customer")
		}
	}

	newCustomer := &Customer{
		ID:    c.ID,
		Email: email,
	}

	result := m.db.WithContext(ctx).Create(newCustomer)
	if result.Error != nil {
		m.logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, extErrors.Wrap(result.Error, "Cannot create a New Customer")
	}

	return newCustomer, nil
}

// GetByID will try to return the customer in the database by id
func (m *Manager) GetByID(ctx context.Context, id string) (*Customer, error) {
	var cust Customer

	result := m.db.WithContext(ctx).First(&cust, "id = ?", id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		m.logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, extErrors.Wrap(result.Error, "Cannot get customer by id")
	}

	return &cust, nil
}

// GetByEmail will try to return the customer in the database by email address
func (m *Manager) GetByEmail(ctx context.Context, email string) (*Customer, error) {
	var cust Customer

	result := m.db.WithContext(ctx).First(&cust, "email = ?", email)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		m.logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, extErrors.Wrap(result.Error, "Cannot get customer by email")
	}

	return &cust, nil
}
