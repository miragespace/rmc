package instance

import (
	"time"

	"github.com/zllovesuki/rmc/usage"
	"gorm.io/gorm"
)

// Instance describes a Minecraft server instance
type Instance struct {
	ID             string    `json:"id" gorm:"primaryKey"`                       // UUID of the server instance. This will also be the name of the Docker container
	CustomerID     string    `json:"customerId" gorm:"index;not null"`           // Corresponds to Stripe's customer ID
	SubscriptionID string    `json:"subscriptionId" gorm:"uniqueIndex;not null"` // Corresponds to Stripe's subscription ID
	HostName       string    `json:"hostName"`                                   // Defines which host the server runs on
	ServerAddr     string    `json:"serverAddr"`                                 // Minecraft server host IP, usually the same as the host's ip
	ServerPort     uint32    `json:"serverPort"`                                 // Minecraft server port
	ServerVersion  string    `json:"serverVersion"`                              // Minecraft server version
	IsJavaEdition  bool      `json:"isJavaEdition"`                              // Minecraft server edition (Java/Bedrock)
	PreviousState  string    `json:"previousState"`                              // See const.go for the list of valid states. This allows for easy error tracking when State is Error
	State          string    `json:"state"`                                      // See const.go for the list of valid states
	Status         string    `json:"status"`                                     // Active/Terminated
	CreatedAt      time.Time `json:"createdAt"`
}

// AfterUpdate will insert an usage report after a control operation was successful using gorm Hooks (https://gorm.io/docs/hooks.html)
func (i *Instance) AfterUpdate(tx *gorm.DB) error {
	var record *usage.Usage
	switch i.State {
	case StateRunning:
		if i.PreviousState == StateStarting || i.PreviousState == StateProvisioning {
			record = &usage.Usage{
				InstanceID: i.ID,
				Action:     usage.Start,
				When:       time.Now(),
			}
		}
	case StateStopped:
		if i.PreviousState == StateStopping {
			record = &usage.Usage{
				InstanceID: i.ID,
				Action:     usage.Stop,
				When:       time.Now(),
			}
		}
	}
	if record == nil {
		return nil
	}
	return tx.Create(record).Error
}
