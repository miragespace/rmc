package instance

// Define the valid state of an instance
// Provisioning -> Running/Error
// Running -> Stopping
// Stopping -> Stopped
// Stopped -> Starting/Removing
// Starting -> Running
// Stopping -> Stopped
// Removing -> Removed/Error
// It is possible that PreviousState is "Running/Stopping" but State is "Running/Stopping"
// That probably means request was sent and completed successful, and background task was able to LambdaUpdate,
// but LambdaUpdate in service was not successful. See task.go for mediation steps
// (I should make a FSM for this)
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
