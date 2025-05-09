package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/terkoizmy/go-blog-api/internal/db"
	"github.com/terkoizmy/go-blog-api/internal/models"
)

type CategoryHandler struct{}

func NewCategoryHandler() *CategoryHandler {
	return &CategoryHandler{}
}

func generateCategorySlug(title string) string {

	slug := strings.ToLower(title)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove special characters
	slug = removeSpecialChars(slug)

	// Remove consecutive hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from beginning and end
	slug = strings.Trim(slug, "-")

	return slug
}

// @Summary Create a new category
// @Description Create a new blog category
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category body models.CategoryRequest true "Category details"
// @Success 201 {object} models.Category
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories [post]
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	// Only admins can create categories
	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	roleStr, ok := role.(string)
	if !ok || roleStr != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req models.CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateCategorySlug(req.Name)
	} else {
		slug = generateCategorySlug(slug)
	}

	// Check if slug already exists
	var existingCategory models.Category
	if result := db.DB.Where("slug = ?", slug).First(&existingCategory); result.RowsAffected > 0 {
		// Add a unique identifier to the slug
		slug = slug + "-" + uuid.New().String()[:8]
	}

	// Create category
	category := models.Category{
		Name: req.Name,
		Slug: slug,
	}

	if result := db.DB.Create(&category); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, category)
}
