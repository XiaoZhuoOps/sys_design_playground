package api

import "github.com/gin-gonic/gin"

// SetupRouter configures the API routes for the application.
func SetupRouter(router *gin.Engine) {
	// Group all API routes under /api
	api := router.Group("/api")
	{
		// Scenario-related endpoints
		scenarios := api.Group("/scenarios")
		{
			scenarios.GET("", ListScenariosHandler)
			scenarios.GET("/:id", GetScenarioHandler)
			scenarios.POST("/:id/actions/:action_id", ExecuteActionHandler)
			scenarios.GET("/:id/state", GetStateHandler)
		}
	}
}