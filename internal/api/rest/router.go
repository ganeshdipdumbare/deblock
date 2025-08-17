package rest

import (
	"log/slog"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	swagFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var validate *validator.Validate

type ErrorResponse struct {
	Message string `json:"message"`
}

func createErrorResponse(c *gin.Context, code int, message string) {
	c.IndentedJSON(code, &ErrorResponse{
		Message: message,
	})
}

func (api *apiDetails) setupRouter() *gin.Engine {
	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode)

	validate = validator.New()
	r := gin.New()

	// Add logging middleware
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health", "/swagger/*any"},
	}))

	// Add recovery middleware to prevent crashes
	r.Use(gin.Recovery())

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowHeaders = append(config.AllowHeaders, "Access-Control-Allow-Origin")
	config.AllowOrigins = []string{"*"}
	r.Use(cors.New(config))

	// Root route for basic info
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "Deblock Transaction Monitor",
			"status":  "running",
		})
	})

	// API V1 group
	apiV1 := r.Group("/api/v1")
	{
		// Swagger documentation
		apiV1.GET("/swagger/*any", ginSwagger.WrapHandler(swagFiles.Handler))

		// Health check
		apiV1.GET("/health", api.health)

		// Transaction monitor routes
		apiV1.POST("/txmonitor/start", api.startTxMonitor)
		apiV1.POST("/txmonitor/stop", api.stopTxMonitor)
	}

	// Log all registered routes
	api.logRoutes(r)

	return r
}

// logRoutes logs all registered routes for debugging
func (api *apiDetails) logRoutes(r *gin.Engine) {
	for _, routeInfo := range r.Routes() {
		api.logger.Info("Registered route",
			slog.String("method", routeInfo.Method),
			slog.String("path", routeInfo.Path),
			slog.String("handler", routeInfo.Handler),
		)
	}
}
