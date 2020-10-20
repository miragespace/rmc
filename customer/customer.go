package customer

// Customer describes a user in RMC
type Customer struct {
	ID    string `json:"id" gorm:"primaryKey"`     // Corresponds to Stripe's customer ID
	Email string `json:"email" gorm:"uniqueIndex"` // User's email address
}
