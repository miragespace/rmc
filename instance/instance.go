package instance

import (
	"time"

	"github.com/zllovesuki/rmc/spec"
)

// Instance describes a Minecraft server instance
type Instance struct {
	ID             string          `json:"id" gorm:"primaryKey"`                       // UUID of the server instance. This will also be the name of the Docker container
	CustomerID     string          `json:"customerId" gorm:"index;not null"`           // Corresponds to Stripe's customer ID
	SubscriptionID string          `json:"subscriptionId" gorm:"uniqueIndex;not null"` // Corresponds to Stripe's subscription ID (soft defined FK to subscription)
	HostName       string          `json:"hostName"`                                   // Defines which host the server runs on (soft defined FK to host)
	Parameters     spec.Parameters `json:"parameters"`                                 // Defines the parameters of the instance
	PreviousState  State           `json:"previousState"`                              // See const.go for the list of valid states
	State          State           `json:"state"`                                      // See const.go for the list of valid states
	Status         Status          `json:"status"`                                     // Active/Terminated
	CreatedAt      time.Time       `json:"createdAt" gorm:"autoCreateTime"`            // When the instance was created
	Histories      []History       `json:"histories"`                                  // State changes throughout instance' life
}

// History describes when an instance's state was changed
type History struct {
	InstanceID string    `json:"-" gorm:"primaryKey;not null"`         // FK to Instance.ID
	Timestamp  time.Time `json:"timestamp" gorm:"primaryKey;not null"` // Timestamp when the Instance.State was changed
	State      State     `json:"state" gorm:"primaryKey;not null"`     // State when the Instance.State was changed
}
