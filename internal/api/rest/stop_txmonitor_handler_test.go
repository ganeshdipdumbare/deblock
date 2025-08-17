package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"deblock/mocks"
)

// TestStopTxMonitor tests the stopTxMonitor handler
func TestStopTxMonitor(t *testing.T) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Successful Transaction Monitor Stop", func(t *testing.T) {
		// Create mock transaction monitor service
		mockTxMonitorService := mocks.NewMockTxMonitorService(ctrl)

		// Expect Stop method to be called with any context
		mockTxMonitorService.EXPECT().
			Stop(gomock.Any()).
			Return(nil)

		// Create API details with mock service and logger
		logger := setupTestLogger()
		apiDetails := &apiDetails{
			logger:  logger,
			service: mockTxMonitorService,
		}

		// Create Gin context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/txmonitor/stop", nil)

		// Call the handler
		apiDetails.stopTxMonitor(c)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code, "HTTP status should be 200 OK")

		// Parse response body
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Should be able to parse response JSON")

		// Verify response contents
		assert.Equal(t, "Transaction monitor stopped successfully", response["message"])
		assert.Equal(t, "stopped", response["status"])
	})

	t.Run("Transaction Monitor Stop Failure", func(t *testing.T) {
		// Create mock transaction monitor service
		mockTxMonitorService := mocks.NewMockTxMonitorService(ctrl)

		// Expect Stop method to be called with any context and return an error
		mockTxMonitorService.EXPECT().
			Stop(gomock.Any()).
			Return(errors.New("stop failed"))

		// Create API details with mock service and logger
		logger := setupTestLogger()
		apiDetails := &apiDetails{
			logger:  logger,
			service: mockTxMonitorService,
		}

		// Create Gin context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/txmonitor/stop", nil)

		// Call the handler
		apiDetails.stopTxMonitor(c)

		// Assert response
		assert.Equal(t, http.StatusInternalServerError, w.Code, "HTTP status should be 500 Internal Server Error")

		// Parse response body
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Should be able to parse response JSON")

		// Verify error response
		assert.Equal(t, "Failed to stop transaction monitor", response["message"], "Error message should match")
	})
}

// TestSetupStopTxMonitorRoutes tests the route setup
func TestSetupStopTxMonitorRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock transaction monitor service
	mockTxMonitorService := mocks.NewMockTxMonitorService(ctrl)

	// Create API details with mock service and logger
	logger := setupTestLogger()
	apiDetails := &apiDetails{
		logger:  logger,
		service: mockTxMonitorService,
	}

	// Setup routes
	router := apiDetails.setupRouter()

	// Check if the route exists
	routes := router.Routes()

	// Find the route
	var stopRoute *gin.RouteInfo
	for _, route := range routes {
		if route.Path == "/api/v1/txmonitor/stop" && route.Method == "POST" {
			stopRoute = &route
			break
		}
	}

	// Assert route exists
	assert.NotNil(t, stopRoute, "Stop transaction monitor route should exist")
}
