package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/terkoizmy/go-blog-api/api/handlers"
	"github.com/terkoizmy/go-blog-api/internal/auth"
)

func SetupPostRoutes(router *gin.Engine) {
	postHandler := handlers.NewPostHandler()

	api := router.Group("/api/v1")
	posts := api.Group("/posts")

	// Public routes
	posts.GET("", postHandler.GetAllPosts)
	posts.GET("/:id", postHandler.GetPostByID)
	posts.GET("/user/:userId", postHandler.GetPostsByUserID)
	posts.GET("/slug/:slug", postHandler.GetPostBySlug)

	// Protected routes
	protected := posts.Group("")
	protected.Use(auth.AuthMiddleware())
	{
		protected.GET("/own", postHandler.GetOwnPosts)
		protected.POST("", postHandler.CreatePost)
		protected.PUT("/:id", postHandler.UpdatePost)
		protected.DELETE("/:id", postHandler.DeletePost)
	}
}
