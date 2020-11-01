package subscription

// State is the custom type to define the current state of a subscription
type State string

// Defining different SubscriptionStates for a Subscription
const (
	StateActive    State = "Active"
	StateInactive  State = "Inactive"
	StatePending   State = "Pending"
	StateCancelled State = "Cancelled"
	StateOverdue   State = "Overdue"
)

// PartType is the custom type to identify what's the Type of this Part in the Plan
type PartType string

// Defining constants
const (
	FixedType    PartType = "Fixed"
	VariableType PartType = "Variable"
)
