package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/terkoizmy/go-blog-api/api/handlers"
	"github.com/terkoizmy/go-blog-api/api/middleware"
)

// SetupUserRoutes configures all the user related routes
func SetupUserRoutes(router *gin.Engine) {
	// Create user handler
	userHandler := handlers.NewUserHandler()

	// Health check route
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "up",
		})
	})

	// Public routes (no authentication required)
	router.POST("/api/v1/register", userHandler.Register)
	router.POST("/api/v1/login", userHandler.Login)

	// Routes that require authentication
	// Create a group with authentication middleware
	authorized := router.Group("/api/v1")
	authorized.Use(middleware.AuthMiddleware())
	{
		// User routes
		users := authorized.Group("/users")
		{
			// Get current user profile
			users.GET("/me", userHandler.GetMe)

			// Admin route to get all users - requires admin role
			users.GET("", middleware.RoleMiddleware("admin"), userHandler.GetAll)

			// User can update their own profile or admin can update any profile
			// The handler itself checks for permissions
			users.PUT("/:id", userHandler.UpdateUser)

			// User can delete their own account or admin can delete any account
			// The handler itself checks for permissions
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}
}
