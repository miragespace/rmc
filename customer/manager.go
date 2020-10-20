package customer

import (
	"errors"

	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/customer"
	"gorm.io/gorm"
)

type Manager struct {
	db *gorm.DB
}

func NewManager(db *gorm.DB) (*Manager, error) {
	if err := db.AutoMigrate(&Customer{}); err != nil {
		return nil, extErrors.Wrap(err, "Cannot initilize customer.Manager")
	}
	return &Manager{
		db: db,
	}, nil
}

func (m *Manager) NewCustomer(email string) (*Customer, error) {
	params := &stripe.CustomerParams{
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

	result := m.db.Create(newCustomer)
	if result.Error != nil {
		return nil, extErrors.Wrap(result.Error, "Cannot create a New Customer")
	}

	return newCustomer, nil
}

func (m *Manager) GetByID(id string) (*Customer, error) {
	var cust Customer

	result := m.db.First(&cust, "id = ?", id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		return nil, extErrors.Wrap(result.Error, "Cannot get customer by id")
	}

	return &cust, nil
}

func (m *Manager) GetByEmail(email string) (*Customer, error) {
	var cust Customer

	result := m.db.First(&cust, "email = ?", email)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		return nil, extErrors.Wrap(result.Error, "Cannot get customer by email")
	}

	return &cust, nil
}
