package customer

import (
	"context"
	"errors"

	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/customer"
	"gorm.io/gorm"
)

// Manager handles the database operations relating to Customers
type Manager struct {
	db *gorm.DB
}

// NewManager returns a new Manager for customers
func NewManager(db *gorm.DB) (*Manager, error) {
	if err := db.AutoMigrate(&Customer{}); err != nil {
		return nil, extErrors.Wrap(err, "Cannot initilize customer.Manager")
	}
	return &Manager{
		db: db,
	}, nil
}

// NewCustomer will create a new customer profile in Stripe and in the database
func (m *Manager) NewCustomer(ctx context.Context, email string) (*Customer, error) {
	params := &stripe.CustomerParams{
		Params: stripe.Params{
			Context: ctx,
		},
		Email: stripe.String(email),
	}

	c, err := customer.New(params)
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot create a new Customer")
	}

	newCustomer := &Customer{
		ID:    c.ID,
		Email: email,
	}

	result := m.db.WithContext(ctx).Create(newCustomer)
	if result.Error != nil {
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
		return nil, extErrors.Wrap(result.Error, "Cannot get customer by email")
	}

	return &cust, nil
}
