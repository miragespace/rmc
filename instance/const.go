package instance

// Define the valid state of an instance
// Provisioning -> Running/Error
// Running -> Stopping/Error
// Stopping -> Stopped/Error
// Stopped -> Starting/Removing
// Starting -> Running/Error
// Stopping -> Stopped/Error
// Removing -> Removed/Error
const (
	StateUnknown      string = "Unknown"
	StateError               = "Error"
	StateProvisioning        = "Provisioning"
	StateStarting            = "Starting"
	StateRunning             = "Running"
	StateStopping            = "Stopping"
	StateStopped             = "Stopped"
	StateRemoving            = "Removing"
	StateRemoved             = "Removed"
)

// Define the valid status of an instance
const (
	StatusActive     string = "Active"
	StatusTerminated        = "Terminated"
)
