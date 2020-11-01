package instance

// State is the custom type to define the current state of an instance
type State string

// Define the valid state of an instance
// Provisioning -> Running/Error
// Running -> Stopping
// Stopping -> Stopped
// Stopped -> Starting/Removing
// Starting -> Running
// Removing -> Removed/Error
// Instance.State should never be "Unknown." Check PreviousState if State is Error
const (
	StateUnknown      State = "Unknown"
	StateError        State = "Error"
	StateProvisioning State = "Provisioning"
	StateStarting     State = "Starting"
	StateRunning      State = "Running"
	StateStopping     State = "Stopping"
	StateStopped      State = "Stopped"
	StateRemoving     State = "Removing"
	StateRemoved      State = "Removed"
)

// Status is the custom type to define the current status of an instance
type Status string

// Define the valid status of an instance
const (
	StatusActive     Status = "Active"
	StatusTerminated Status = "Terminated"
)
