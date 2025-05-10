package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/terkoizmy/go-blog-api/internal/db"
	"github.com/terkoizmy/go-blog-api/internal/models"
)

// PostHandler handles post-related routes
type PostHandler struct{}

func NewPostHandler() *PostHandler {
	return &PostHandler{}
}

// Helper for generate slog from title
func generateSlug(title string) string {

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

func removeSpecialChars(input string) string {
	// Replace everything except letters, numbers, and space
	re := regexp.MustCompile(`[^a-zA-Z0-9\s-]+`)
	return re.ReplaceAllString(input, "")
}

// @Summary Create a new post
// @Description Create a new blog post
// @Tags posts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param post body models.PostRequest true "Post details"
// @Success 201 {object} models.Post
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /posts [post]
func (h *PostHandler) CreatePost(c *gin.Context) {
	var req models.PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	authorID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Title)
	} else {
		slug = generateSlug(slug)
	}

	// Check if slug already exists
	var existingPost models.Post
	if result := db.DB.Where("slug = ?", slug).First(&existingPost); result.RowsAffected > 0 {
		// Add a unique identifier to the slug
		slug = slug + "-" + uuid.New().String()[:8]
	}

	// Set default status if not provided
	status := req.Status
	if status == "" {
		status = "draft"
	}

	// Create post
	post := models.Post{
		Title:    req.Title,
		Content:  req.Content,
		Slug:     slug,
		Status:   status,
		AuthorID: authorID,
	}

	// If status is "published", set PublishedAt to nncurrent time
	if status == "published" {
		now := time.Now()
		post.PublishedAt = &now
	}

	if result := db.DB.Create(&post); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create post"})
		return
	}

	// Add categories if provided
	if len(req.CategoryIDs) > 0 {
		for _, categoryID := range req.CategoryIDs {
			// Check if category exists
			var category models.Category
			if result := db.DB.Where("id = ?", categoryID).First(&category); result.Error != nil {
				// Skip invalid categories
				continue
			}

			// Add association
			db.DB.Model(&post).Association("Categories").Append(&category)
		}
	}

	// Load author details
	var author models.User
	db.DB.Where("id = ?", authorID).First(&author)
	post.Author = author
	post.Author.Password = "" // Don't return password

	// Load categories
	db.DB.Model(&post).Association("Categories").Find(&post.Categories)

	c.JSON(http.StatusCreated, post)

}

// @Summary Get all posts
// @Description Get all blog posts
// @Tags posts
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param status query string false "Filter by status"
// @Success 200 {array} models.Post
// @Failure 500 {object} map[string]string
// @Router /posts [get]
func (h *PostHandler) GetAllPosts(c *gin.Context) {
	// Pagination
	page := 1
	if pageQuery, exists := c.GetQuery("page"); exists {
		if _, err := c.Get(pageQuery); err {
			page = 1
		}
	}

	limit := 10
	// if limitQuery, exists := c.GetQuery("limit"); exists {
	// 	if val, err := c.GetInt(limitQuery); err == nil {
	// 		limit = val
	// 	}
	// }

	if limitStr, exists := c.GetQuery("limit"); exists {
		if limitVal, err := strconv.Atoi(limitStr); err == nil {
			limit = limitVal
		}
	}

	offset := (page - 1) * limit

	// Status filter
	status := c.Query("status")

	var posts []models.Post
	query := db.DB.Offset(offset).Limit(limit).Preload("Author").Preload("Categories")

	// Apply status filter if provided
	if status != "" {
		query = query.Where("status = ?", status)
	} else {
		// By default, only show published posts to public
		query = query.Where("status = ?", "published")
	}

	// Execute query
	if result := query.Find(&posts); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get posts"})
		return
	}

	// Clean up author passwords
	for i := range posts {
		posts[i].Author.Password = ""
		posts[i].Author.Role = ""
	}

	c.JSON(http.StatusOK, posts)
}

// @Summary Get post by ID
// @Description Get a post by its ID
// @Tags posts
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Success 200 {object} models.Post
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /posts/{id} [get]
func (h *PostHandler) GetPostByID(c *gin.Context) {
	id := c.Param("id")

	// Parse the UUID
	postUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID format"})
		return
	}

	var post models.Post
	if result := db.DB.Preload("Author").Preload("Categories").Preload("Comments.Author").Where("id = ?", postUUID).First(&post); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	// Check if the post is published or the user is the author or admin
	if post.Status != "published" {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not published yet from author"})
		return
		// userID, exists := c.Get("userID")
		// userRole, roleExists := c.Get("role")

		// // If not authenticated or not the author or admin, return 404
		// if !exists || !roleExists {
		// 	c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		// 	return
		// }

		// authorID, idOk := userID.(uuid.UUID)
		// roleStr, roleOk := userRole.(string)

		// if !idOk || !roleOk || (post.AuthorID != authorID && roleStr != "admin") {
		// 	c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		// 	return
		// }
	}

	// Clean up sensitive information
	post.Author.Password = ""
	post.Author.Role = ""
	for i := range post.Comments {
		post.Comments[i].Author.Password = ""
	}

	c.JSON(http.StatusOK, post)
}

// @Summary Get own posts
// @Description Get all own posts
// @Tags posts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Post
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /posts/own [get]
func (h *PostHandler) GetOwnPosts(c *gin.Context) {
	ownID, exists := c.Get("userID")
	fmt.Println("ownID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	fmt.Println("masuk")
	userID, ok := ownID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	var posts []models.Post
	fmt.Println("masuk 1")
	if result := db.DB.Preload("Categories").Where("author_id = ?", userID).Find(&posts); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get posts"})
		return
	}
	fmt.Println("masuk 2")
	// Clean up author passwords
	for i := range posts {
		posts[i].Author.Password = ""
	}

	c.JSON(http.StatusOK, posts)
}

// @Summary Get post by USER ID
// @Description Get a USER post by its USER ID
// @Tags posts
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} models.Post
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /posts/user/{userId} [get]
func (h *PostHandler) GetPostsByUserID(c *gin.Context) {
	userIDParam := c.Param("userId")

	// Parse the UUID
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid User ID format"})
		return
	}

	var posts []models.Post
	if result := db.DB.Preload("Author").Preload("Categories").Preload("Comments.Author").Where("author_id = ?", userID).Where("status = ?", "published").Find(&posts); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Failet to get posts"})
		return
	}

	// Clean up sensitive information
	for i := range posts {
		posts[i].Author.Password = ""
		posts[i].Author.Role = ""
	}

	c.JSON(http.StatusOK, posts)
}

// @Summary Get post by slug
// @Description Get a post by its slug
// @Tags posts
// @Accept json
// @Produce json
// @Param slug path string true "Post Slug"
// @Success 200 {object} models.Post
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /posts/slug/{slug} [get]
func (h *PostHandler) GetPostBySlug(c *gin.Context) {
	slug := c.Param("slug")

	var post models.Post
	if result := db.DB.Preload("Author").Preload("Categories").Preload("Comments.Author").Where("slug = ?", slug).First(&post); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	// // Check if the post is published or the user is the author or admin
	// if post.Status != "published" {
	// 	userID, exists := c.Get("userID")
	// 	userRole, roleExists := c.Get("role")

	// 	// If not authenticated or not the author or admin, return 404
	// 	if !exists || !roleExists {
	// 		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
	// 		return
	// 	}

	// 	authorID, idOk := userID.(uuid.UUID)
	// 	roleStr, roleOk := userRole.(string)

	// 	if !idOk || !roleOk || (post.AuthorID != authorID && roleStr != "admin") {
	// 		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
	// 		return
	// 	}
	// }

	// Clean up sensitive information
	post.Author.Password = ""
	post.Author.Role = ""
	for i := range post.Comments {
		post.Comments[i].Author.Password = ""
	}

	c.JSON(http.StatusOK, post)
}

// @Summary Update post
// @Description Update a post
// @Tags posts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Post ID"
// @Param post body models.PostRequest true "Updated post details"
// @Success 200 {object} models.Post
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /posts/{id} [put]
func (h *PostHandler) UpdatePost(c *gin.Context) {
	id := c.Param("id")

	// Parse the UUID
	postUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID format"})
		return
	}

	var post models.Post
	if result := db.DB.Where("id = ?", postUUID).First(&post); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	// Check if user is the author or admin
	userID, exists := c.Get("userID")
	userRole, roleExists := c.Get("role")

	if !exists || !roleExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	authorID, idOk := userID.(uuid.UUID)
	roleStr, roleOk := userRole.(string)

	if !idOk || !roleOk || (post.AuthorID != authorID && roleStr != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req models.PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.Title != "" {
		post.Title = req.Title
	}

	if req.Content != "" {
		post.Content = req.Content
	}

	// Update slug if provided, otherwise generate from title
	if req.Slug != "" {
		post.Slug = generateSlug(req.Slug)
	} else if req.Title != "" {
		post.Slug = generateSlug(req.Title)
	}

	// Check if slug already exists
	var existingPost models.Post
	if result := db.DB.Where("slug = ? AND id != ?", post.Slug, postUUID).First(&existingPost); result.RowsAffected > 0 {
		// Add a unique identifier to the slug
		post.Slug = post.Slug + "-" + uuid.New().String()[:8]
	}

	// Update status if provided
	if req.Status != "" && req.Status != post.Status {
		post.Status = req.Status

		// If changing to published, set published time
		if req.Status == "published" && post.PublishedAt == nil {
			now := time.Now()
			post.PublishedAt = &now
		}
	}

	// Save the post
	if result := db.DB.Save(&post); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update post"})
		return
	}

	// Update categories if provided
	if len(req.CategoryIDs) > 0 {
		// Clear existing categories
		db.DB.Model(&post).Association("Categories").Clear()

		// Add new categories
		for _, categoryID := range req.CategoryIDs {
			// Check if category exists
			var category models.Category
			if result := db.DB.Where("id = ?", categoryID).First(&category); result.Error != nil {
				// Skip invalid categories
				continue
			}

			// Add association
			db.DB.Model(&post).Association("Categories").Append(&category)
		}
	}

	// Load updated post with associations
	db.DB.Preload("Author").Preload("Categories").Where("id = ?", postUUID).First(&post)

	// Clean up sensitive information
	post.Author.Password = ""
	post.Author.Role = ""
	c.JSON(http.StatusOK, post)
}

// @Summary Delete post
// @Description Delete a post
// @Tags posts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Post ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /posts/{id} [delete]
func (h *PostHandler) DeletePost(c *gin.Context) {
	id := c.Param("id")

	// Parse the UUID
	postUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID format"})
		return
	}

	var post models.Post
	if result := db.DB.Where("id = ?", postUUID).First(&post); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	// Check if user is the author or admin
	userID, exists := c.Get("userID")
	userRole, roleExists := c.Get("role")

	if !exists || !roleExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	authorID, idOk := userID.(uuid.UUID)
	roleStr, roleOk := userRole.(string)

	if !idOk || !roleOk || (post.AuthorID != authorID && roleStr != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	// Delete the post (this will use soft delete due to GORM's DeletedAt field)
	if result := db.DB.Delete(&post); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "post deleted successfully"})
}
