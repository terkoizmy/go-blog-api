package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/terkoizmy/go-blog-api/api/handlers"
	"github.com/terkoizmy/go-blog-api/internal/auth"
)

func SetupCommentRoutes(router *gin.Engine) {
	commentHandler := handlers.NewCommentHandler()

	api := router.Group("/api/v1")
	comment := api.Group("/comment")

	// Public routes
	comment.GET("/posts/:postId", commentHandler.GetAllCommentsFromPostId)
	comment.GET("/:id", commentHandler.GetCommentById)
	// comment.GET("/slug/:slug", categoryHandler.GetCategoryBySlug)

	// Protected routes
	protected := comment.Group("")
	protected.Use(auth.AuthMiddleware())
	{
		protected.POST("/posts/:postId", commentHandler.CreateComment)
		protected.PUT("/:id", commentHandler.UpdateComment)
		protected.DELETE("/:id", commentHandler.DeleteComment)
	}

}
