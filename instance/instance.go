package instance

// Instance describes a Minecraft server instance
type Instance struct {
	ID             string `json:"id" gorm:"primaryKey"`          // UUID of the server instance. This will also be the name of the Docker container
	CustomerID     string `json:"customerId" gorm:"uniqueIndex"` // Corresponds to Stripe's customer ID
	SubscriptionID string `json:"subscriptionId"`                // Corresponds to Stripe's subscription ID
	Host           string `json:"host"`                          // Defines which host the server runs on
	ServerAddr     string `json:"serverAddr"`                    // Minecraft server host IP
	ServerPort     int    `json:"serverPort"`                    // Minecraft server port
	ServerVersion  string `json:"version"`                       // Minecraft server version
	IsJavaEdition  bool   `json:"isJavaEdition"`                 // Minecraft server edition (Java/Bedrock)
	State          string `json:"state"`                         // Starting/Stopping/Started/Stopped/Unknown/Provisioning
	Status         string `json:"status"`                        // Active/Terminated
}
