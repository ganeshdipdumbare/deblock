package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// stopTxMonitor godoc
// @Summary Stop transaction monitor
// @Description Stop the transaction monitor
// @Tags txmonitor
// @Accept json
// @Produce json
// @Success 200 {object} string "ok"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /txmonitor/stop [post]
func (api *apiDetails) stopTxMonitor(c *gin.Context) {
	// Create a context for stopping the transaction monitor
	ctx := c.Request.Context()

	// Log the stop attempt
	api.logger.Info("Attempting to stop transaction monitor")

	// Stop the transaction monitor
	if err := api.service.Stop(ctx); err != nil {
		// If there's an error stopping the transaction monitor, return a 500 error
		api.logger.Error("Failed to stop transaction monitor",
			"error", err,
			"service_type", api.service,
		)
		createErrorResponse(c, http.StatusInternalServerError, "Failed to stop transaction monitor")
		return
	}

	// Log successful stop
	api.logger.Info("Transaction monitor stopped successfully")

	// Respond with success
	c.JSON(http.StatusOK, gin.H{
		"message": "Transaction monitor stopped successfully",
		"status":  "stopped",
	})
}
