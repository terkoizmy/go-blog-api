package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/terkoizmy/go-blog-api/api/handlers"
	"github.com/terkoizmy/go-blog-api/internal/auth"
)

func SetupCategoryRoutes(router *gin.Engine) {
	categoryHandler := handlers.NewCategoryHandler()

	api := router.Group("/api/v1")
	categories := api.Group("/categories")

	// Public routes
	categories.GET("", categoryHandler.GetAllCategories)
	categories.GET("/:id", categoryHandler.GetCategoryByID)
	categories.GET("/slug/:slug", categoryHandler.GetCategoryBySlug)

	// Protected routes
	protected := categories.Group("")
	protected.Use(auth.AuthMiddleware())
	{
		protected.POST("", categoryHandler.CreateCategory)
		protected.PUT("/:id", categoryHandler.UpdateCategory)
		protected.DELETE("/:id", categoryHandler.DeleteCategory)
	}

}
