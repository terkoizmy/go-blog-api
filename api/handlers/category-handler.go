package handlers

import (
	"net/http"
	"strconv"
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

// @Summary Get all categories
// @Description Get all blog categories
// @Tags categories
// @Accept json
// @Produce json
// @Success 200 {array} models.Category
// @Failure 500 {object} map[string]string
// @Router /categories [get]
func (h *CategoryHandler) GetAllCategories(c *gin.Context) {
	var categories []models.Category
	if result := db.DB.Find(&categories); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get categories"})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// @Summary Get category by ID
// @Description Get a category by its ID
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} models.Category
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories/{id} [get]
func (h *CategoryHandler) GetCategoryByID(c *gin.Context) {
	id := c.Param("id")

	// Parse the UUID
	categoryUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category ID format"})
		return
	}

	var category models.Category
	if result := db.DB.Where("id = ?", categoryUUID).First(&category); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}

	c.JSON(http.StatusOK, category)
}

// @Summary Get category by slug
// @Description Get a category by its slug
// @Tags categories
// @Accept json
// @Produce json
// @Param slug path string true "Category Slug"
// @Success 200 {object} models.Category
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories/slug/{slug} [get]
func (h *CategoryHandler) GetCategoryBySlug(c *gin.Context) {
	slug := c.Param("slug")

	var category models.Category
	if result := db.DB.Where("slug = ?", slug).First(&category); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}

	c.JSON(http.StatusOK, category)
}

// @Summary Get posts by category
// @Description Get all posts in a specific category
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {array} models.Post
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories/{id}/posts [get]
func (h *CategoryHandler) GetPostsByCategory(c *gin.Context) {
	id := c.Param("id")

	// Parse the UUID
	categoryUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category ID format"})
		return
	}

	// Check if category exists

	var category models.Category
	if result := db.DB.Where("id = ?", categoryUUID).First(&category); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}

	// Pagination
	page := 1
	if pageQuery := c.Query("page"); pageQuery != "" {
		if val, err := strconv.Atoi(pageQuery); err == nil && val > 0 {
			page = val
		}
	}

	limit := 10
	if limitQuery := c.Query("limit"); limitQuery != "" {
		if val, err := strconv.Atoi(limitQuery); err == nil && val > 0 {
			limit = val
		}
	}

	offset := (page - 1) * limit

	// Get posts by category with pagination
	var posts []models.Post
	if result := db.DB.Joins("JOIN post_categories ON posts.id = post_categories.post_id").
		Where("post_categories.category_id = ? AND posts.status = ?", categoryUUID, "published").
		Offset(offset).Limit(limit).Preload("Author").Preload("Categories").
		Find(&posts); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get posts"})
		return
	}

	// Clean up sensitive information
	for i := range posts {
		posts[i].Author.Password = ""
	}

	c.JSON(http.StatusOK, posts)

}

// @Summary Update category
// @Description Update a category
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Category ID"
// @Param category body models.CategoryRequest true "Updated category details"
// @Success 200 {object} models.Category
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	// Only admins can update categories
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

	id := c.Param("id")

	// Parse the UUID
	categoryUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category ID format"})
		return
	}

	var category models.Category
	if result := db.DB.Where("id = ?", categoryUUID).First(&category); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}

	var req models.CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.Name != "" {
		category.Name = req.Name
	}

	// Update slug if provided, otherwise generate from name
	if req.Slug != "" {
		category.Slug = generateCategorySlug(req.Slug)
	} else if req.Name != "" {
		category.Slug = generateCategorySlug(req.Name)
	}

	// Check if slug already exists
	var existingCategory models.Category
	if result := db.DB.Where("slug = ? AND id != ?", category.Slug, categoryUUID).First(&existingCategory); result.RowsAffected > 0 {
		// Add a unique identifier to the slug
		category.Slug = category.Slug + "-" + uuid.New().String()[:8]
	}

	// Save the category
	if result := db.DB.Save(&category); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update category"})
		return
	}

	c.JSON(http.StatusOK, category)
}

// @Summary Delete category
// @Description Delete a category
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Category ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	// Only admins can delete categories
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

	id := c.Param("id")

	// Parse the UUID
	categoryUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category ID format"})
		return
	}

	var category models.Category
	if result := db.DB.Where("id = ?", categoryUUID).First(&category); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}

	// Remove category associations from posts first
	db.DB.Model(&category).Association("Posts").Clear()

	// Delete the category
	if result := db.DB.Delete(&category); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "category deleted successfully"})
}
