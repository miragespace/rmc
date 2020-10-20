package instance

// Instance describes a Minecraft server instance
type Instance struct {
	ID             string `json:"id"`             // UUID of the server instance
	CustomerID     string `json:"customerId"`     // Corresponds to Stripe's customer ID
	SubscriptionID string `json:"subscriptionId"` // Corresponds to Stripe's subscription ID
	ServerAddr     string `json:"serverAddr"`     // Minecraft server host IP
	ServerPort     int    `json:"serverPort"`     // Minecraft server port
	State          string `json:"state"`          // Starting/Stopping/Started/Stopped/Unknown
	Status         string `json:"status"`         // Active/Terminated
}
