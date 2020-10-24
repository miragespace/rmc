package instance

// Define the valid state of an instance
const (
	StateUnknown      string = "Unknown"
	StateError               = "Error"
	StateRunning             = "Running"
	StateStarting            = "Starting"
	StateStopped             = "Stopped"
	StateStopping            = "Stopping"
	StateProvisioning        = "Provisioning"
)

// Define the valid status of an instance
const (
	StatusActive     string = "Active"
	StatusTerminated        = "Terminated"
)
