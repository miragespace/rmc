package instance

// Define the valid state of an instance
// Provisioning -> Running/Error
// Running -> Stopping
// Stopping -> Stopped
// Stopped -> Starting/Removing
// Starting -> Running
// Removing -> Removed/Error
// Instance.State should never be "Unknown." Check PreviousState if State is Error
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
