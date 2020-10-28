package instance

import (
	"time"

	"github.com/lithammer/shortuuid/v3"
	"github.com/zllovesuki/rmc/spec"
	"gorm.io/gorm"
)

// Instance describes a Minecraft server instance
type Instance struct {
	ID             string          `json:"id" gorm:"primaryKey"`                       // UUID of the server instance. This will also be the name of the Docker container
	CustomerID     string          `json:"customerId" gorm:"index;not null"`           // Corresponds to Stripe's customer ID
	SubscriptionID string          `json:"subscriptionId" gorm:"uniqueIndex;not null"` // Corresponds to Stripe's subscription ID (soft defined FK to subscription)
	HostName       string          `json:"hostName"`                                   // Defines which host the server runs on (soft defined FK to host)
	Parameters     spec.Parameters `json:"parameters"`                                 // Defines the parameters of the instance
	PreviousState  string          `json:"previousState"`                              // See const.go for the list of valid states
	State          string          `json:"state"`                                      // See const.go for the list of valid states
	Status         string          `json:"status"`                                     // Active/Terminated
	CreatedAt      time.Time       `json:"createdAt" gorm:"autoCreateTime"`            // When the instance was created
	Histories      []History       `json:"histories"`                                  // State changes throughout instance' life
}

// History describes when an instance's state was changed
type History struct {
	ID         string    `json:"-" gorm:"primaryKey"`                                            // ShortUUID of the history record
	InstanceID string    `json:"-" gorm:"not null;index:idx_histories_accounting"`               // FK to Instance.ID
	State      string    `json:"state" gorm:"not null"`                                          // State when the Instance.State was changed
	When       time.Time `json:"when" gorm:"autoCreateTime;not null"`                            // Timestamp when the Instance.State was changed
	Accounted  bool      `json:"-" gorm:"not null;default:false;index:idx_histories_accounting"` // Used for accounting purpose. True if the usage was accounted for and submitted for billing
}

// AfterSave will insert a history when State changes (https://gorm.io/docs/hooks.html)
func (i *Instance) AfterSave(tx *gorm.DB) error {
	if i.PreviousState != i.State {
		return tx.Create(&History{
			ID:         shortuuid.New(),
			InstanceID: i.ID,
			State:      i.State,
		}).Error
	}
	return nil
}
