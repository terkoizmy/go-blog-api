package main

import (
	"log"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/terkoizmy/go-blog-api/api/routes"
	"github.com/terkoizmy/go-blog-api/config"
	_ "github.com/terkoizmy/go-blog-api/docs" // Import docs
	"github.com/terkoizmy/go-blog-api/internal/db"
	"github.com/terkoizmy/go-blog-api/internal/models"
)

// @title           Blog API
// @version         1.0
// @description     A RESTful API for a blog application built with Go and Gin
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.yourwebsite.com/support
// @contact.email  support@yourwebsite.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and the JWT token.
func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set gin mode based on config
	gin.SetMode(cfg.GinMode)

	// Initialize database
	db.InitDB(cfg)

	// Auto migrate the schema
	db.DB.AutoMigrate(&models.User{}, &models.Post{}, &models.Category{}, &models.Comment{})

	// Initialize router
	router := gin.Default()

	// Setup routes
	routes.SetupUserRoutes(router)
	routes.SetupPostRoutes(router)
	routes.SetupCategoryRoutes(router)
	routes.SetupCommentRoutes(router)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Start server
	log.Printf("Server running on port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}
