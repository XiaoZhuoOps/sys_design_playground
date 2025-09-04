package registry

import (
	"fmt"
	"sync"

	"SYS_DESIGN_PLAYGROUND/pkg/scenario"
)

var (
	// scenarios is a thread-safe map to store all registered scenario implementations.
	scenarios = make(map[string]scenario.Scenario)
	// lock is used to protect access to the scenarios map.
	lock = &sync.RWMutex{}
)

// Register adds a new scenario to the registry.
// It will panic if a scenario with the same ID is already registered,
// ensuring that all scenario IDs are unique at startup.
func Register(s scenario.Scenario) {
	lock.Lock()
	defer lock.Unlock()

	id := s.ID()
	if _, exists := scenarios[id]; exists {
		panic(fmt.Sprintf("scenario with ID '%s' is already registered", id))
	}
	scenarios[id] = s
}

// GetScenario retrieves a single scenario from the registry by its ID.
// It returns the scenario and a boolean indicating if it was found.
func GetScenario(id string) (scenario.Scenario, bool) {
	lock.RLock()
	defer lock.RUnlock()

	s, ok := scenarios[id]
	return s, ok
}

// ListScenarios returns a slice of all registered scenarios.
func ListScenarios() []scenario.Scenario {
	lock.RLock()
	defer lock.RUnlock()

	list := make([]scenario.Scenario, 0, len(scenarios))
	for _, s := range scenarios {
		list = append(list, s)
	}
	return list
}

// InitializeAll initializes all registered scenarios.
// This should be called once at application startup.
func InitializeAll() error {
	lock.RLock()
	defer lock.RUnlock()

	fmt.Printf("Initializing %d scenarios...\n", len(scenarios))
	for id, s := range scenarios {
		if err := s.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize scenario '%s': %w", id, err)
		}
		fmt.Printf(" - Scenario '%s' initialized successfully.\n", id)
	}
	return nil
}