package api

import (
	"net/http"

	"SYS_DESIGN_PLAYGROUND/internal/registry"
	"github.com/gin-gonic/gin"
)

// ListScenariosHandler handles the GET /api/scenarios endpoint.
// It returns a list of metadata for all registered scenarios.
func ListScenariosHandler(c *gin.Context) {
	allScenarios := registry.ListScenarios()
	// We only want to return the metadata, not the full scenario object.
	type scenarioMetadata struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Category string `json:"category"`
	}
	metadata := make([]scenarioMetadata, len(allScenarios))
	for i, s := range allScenarios {
		metadata[i] = scenarioMetadata{
			ID:       s.ID(),
			Title:    s.Name(),
			Category: s.Category(),
		}
	}
	c.JSON(http.StatusOK, metadata)
}

// GetScenarioHandler handles the GET /api/scenarios/:id endpoint.
// It returns the full configuration for a single scenario.
func GetScenarioHandler(c *gin.Context) {
	scenarioID := c.Param("id")
	s, ok := registry.GetScenario(scenarioID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scenario not found"})
		return
	}

	// Return the full scenario configuration.
	type scenarioConfig struct {
		ID                  string                        `json:"id"`
		Title               string                        `json:"title"`
		Category            string                        `json:"category"`
		ProblemDescription  string                        `json:"problem_description"`
		SolutionDescription string                        `json:"solution_description"`
		DeepDiveLink        string                        `json:"deep_dive_link"`
		Actions             interface{}                   `json:"actions"`
		DashboardComponents interface{}                   `json:"dashboard_components"`
	}

	config := scenarioConfig{
		ID:                  s.ID(),
		Title:               s.Name(),
		Category:            s.Category(),
		ProblemDescription:  s.ProblemDescription(),
		SolutionDescription: s.SolutionDescription(),
		DeepDiveLink:        s.DeepDiveLink(),
		Actions:             s.Actions(),
		DashboardComponents: s.DashboardComponents(),
	}

	c.JSON(http.StatusOK, config)
}

// ExecuteActionHandler handles the POST /api/scenarios/:id/actions/:action_id endpoint.
// It executes a specific action for a given scenario.
func ExecuteActionHandler(c *gin.Context) {
	scenarioID := c.Param("id")
	actionID := c.Param("action_id")

	s, ok := registry.GetScenario(scenarioID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scenario not found"})
		return
	}

	// For now, we don't handle request body params.
	result, err := s.ExecuteAction(actionID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Action executed successfully.",
		"result":  result,
	})
}

// GetStateHandler handles the GET /api/scenarios/:id/state endpoint.
// It fetches the current state for a scenario's dashboard.
func GetStateHandler(c *gin.Context) {
	scenarioID := c.Param("id")
	s, ok := registry.GetScenario(scenarioID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scenario not found"})
		return
	}

	state, err := s.FetchState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}