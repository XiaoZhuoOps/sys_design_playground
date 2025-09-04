package scenario

// Action represents a user-triggerable event in a scenario.
type Action struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DashboardComponent defines a piece of state to be visualized on the frontend.
type DashboardComponent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // e.g., "key_value", "log_stream", "topic_messages"
}

// Scenario is the interface that every problem demonstration must implement.
// It defines the metadata, actions, and state management for a specific backend problem.
type Scenario interface {
	// Metadata returns the static information about the scenario.
	ID() string
	Name() string
	Category() string
	ProblemDescription() string
	SolutionDescription() string
	DeepDiveLink() string
	Actions() []Action
	DashboardComponents() []DashboardComponent

	// Initialize is called once when the application starts.
	// It's used to set up any required resources like database tables or initial data.
	Initialize() error

	// ExecuteAction runs a specific action defined by the scenario.
	// It takes an actionID and optional parameters.
	ExecuteAction(actionID string, params map[string]interface{}) (interface{}, error)

	// FetchState retrieves the current state of all dashboard components for the scenario.
	// The keys of the returned map should match the IDs of the DashboardComponents.
	FetchState() (map[string]interface{}, error)
}