package instance

import "time"

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
	State          string    `json:"state"`                                      // Starting/Stopping/Started/Stopped/Unknown/Provisioning/Error
	Status         string    `json:"status"`                                     // Active/Terminated
	CreatedAt      time.Time `json:"createdAt"`
}
