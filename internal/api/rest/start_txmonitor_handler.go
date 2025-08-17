package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// startTxMonitor godoc
// @Summary Start transaction monitor
// @Description Start the transaction monitor
// @Tags txmonitor
// @Accept json
// @Produce json
// @Success 200 {object} string "ok"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /txmonitor/start [post]
func (api *apiDetails) startTxMonitor(c *gin.Context) {
	// Create a context for starting the transaction monitor
	ctx := c.Request.Context()

	// Log the start attempt
	api.logger.Info("Attempting to start transaction monitor")

	// Start the transaction monitor
	if err := api.service.Start(ctx); err != nil {
		// If there's an error starting the transaction monitor, return a 500 error
		api.logger.Error("Failed to start transaction monitor",
			"error", err,
			"service_type", api.service,
		)
		createErrorResponse(c, http.StatusInternalServerError, "Failed to start transaction monitor")
		return
	}

	// Log successful start
	api.logger.Info("Transaction monitor started successfully")

	// Respond with success
	c.JSON(http.StatusOK, gin.H{
		"message": "Transaction monitor started successfully",
		"status":  "running",
	})
}
