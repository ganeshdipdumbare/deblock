package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"deblock/mocks"
)

// setupTestLogger creates a test logger
func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// TestStartTxMonitor tests the startTxMonitor handler
func TestStartTxMonitor(t *testing.T) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Successful Transaction Monitor Start", func(t *testing.T) {
		// Create mock transaction monitor service
		mockTxMonitorService := mocks.NewMockTxMonitorService(ctrl)

		// Expect Start method to be called with any context
		mockTxMonitorService.EXPECT().
			Start(gomock.Any()).
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
		c.Request, _ = http.NewRequest(http.MethodPost, "/txmonitor/start", nil)

		// Call the handler
		apiDetails.startTxMonitor(c)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code, "HTTP status should be 200 OK")

		// Parse response body
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Should be able to parse response JSON")

		// Verify response contents
		assert.Equal(t, "Transaction monitor started successfully", response["message"])
		assert.Equal(t, "running", response["status"])
	})

	t.Run("Transaction Monitor Start Failure", func(t *testing.T) {
		// Create mock transaction monitor service
		mockTxMonitorService := mocks.NewMockTxMonitorService(ctrl)

		// Expect Start method to be called with any context and return an error
		mockTxMonitorService.EXPECT().
			Start(gomock.Any()).
			Return(errors.New("start failed"))

		// Create API details with mock service and logger
		logger := setupTestLogger()
		apiDetails := &apiDetails{
			logger:  logger,
			service: mockTxMonitorService,
		}

		// Create Gin context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/txmonitor/start", nil)

		// Call the handler
		apiDetails.startTxMonitor(c)

		// Assert response
		assert.Equal(t, http.StatusInternalServerError, w.Code, "HTTP status should be 500 Internal Server Error")

		// Parse response body
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Should be able to parse response JSON")

		// Verify error response
		assert.Equal(t, "Failed to start transaction monitor", response["message"], "Error message should match")
	})
}

// TestSetupStartTxMonitorRoutes tests the route setup
func TestSetupStartTxMonitorRoutes(t *testing.T) {
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
	var startRoute *gin.RouteInfo
	for _, route := range routes {
		if route.Path == "/api/v1/txmonitor/start" && route.Method == "POST" {
			startRoute = &route
			break
		}
	}

	// Assert route exists
	assert.NotNil(t, startRoute, "Start transaction monitor route should exist")
}
