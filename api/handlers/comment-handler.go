package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/terkoizmy/go-blog-api/internal/db"
	"github.com/terkoizmy/go-blog-api/internal/models"
)

// CommentHandler handles comment-related routes
type CommentHandler struct{}

// NewCommentHandler creates a new CommentHandlerz
func NewCommentHandler() *CommentHandler {
	return &CommentHandler{}
}

// @Summary Create a new comment
// @Description Create a new comment on a post
// @Tags comments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param postId path string true "Post ID"
// @Param comment body models.CommentRequest true "Comment details"
// @Success 201 {object} models.Comment
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /comment/posts/{postId} [post]
func (h *CommentHandler) CreateComment(c *gin.Context) {
	postID := c.Param("postId")
	// Parse the UUID
	postUUID, err := uuid.Parse(postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID format"})
		return
	}

	// Check if post exists and is published
	var post models.Post
	if result := db.DB.Where("id = ?", postUUID).First(&post); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	// Only allow comments on published posts
	if post.Status != "published" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot comment on unpublished posts"})
		return
	}

	// Get user ID from context
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

	var req models.CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create comment
	comment := models.Comment{
		Content:  req.Content,
		PostID:   postUUID,
		AuthorID: authorID,
		ParentID: req.ParentID,
	}

	// If ParentID is provided, check if parent comment exists
	if req.ParentID != nil {
		var parentComment models.Comment
		if result := db.DB.Where("id = ?", *req.ParentID).First(&parentComment); result.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parent comment not found"})
			return
		}

		// Check if parent comment belongs to the same
		if parentComment.PostID != postUUID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parent comment does not belong to the same post"})
			return
		}
	}

	// Don't return the password
	// comment.Author.Password = ""

	if err := db.DB.Preload("Author").Preload("Parent").Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, comment)

}

// @Summary Get comments by POST ID
// @Description Get a Comments post by its post ID
// @Tags comments
// @Accept json
// @Produce json
// @Param postId path string true "Post Id"
// @Success 200 {object} models.Comment
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /comment/posts/{postId} [get]
func (h *CommentHandler) GetAllCommentsFromPostId(c *gin.Context) {
	postId := c.Param("postId")

	// Parse the UUID
	postUUID, err := uuid.Parse(postId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID format"})
		return
	}

	var comments []models.Comment
	if result := db.DB.Preload("Author").Preload("Parent").Where("post_id = ?", postUUID).Find(&comments); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

// @Summary Get comment by comment ID
// @Description Get a Comment by its id
// @Tags comments
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Success 200 {object} models.Comment
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /comment/{id} [get]
func (h *CommentHandler) GetCommentById(c *gin.Context) {
	commentID := c.Param("id")

	// Parse the UUID
	commentUUID, err := uuid.Parse(commentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment ID format"})
		return
	}

	var comment models.Comment
	if result := db.DB.Preload("Author").Preload("Parent").Preload("Replies").Where("id = ?", commentUUID).First(&comment); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}

	c.JSON(http.StatusOK, comment)

}

// @Summary Update Comment
// @Description Update a comment
// @Tags comments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Comment ID"
// @Param comment body models.CommentRequest true "Updated comment details"
// @Success 200 {object} models.Comment
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /comment/{id} [put]
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	id := c.Param("id")

	// Parse the UUID
	commentUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment ID format"})
		return
	}

	var comment models.Comment
	if result := db.DB.Preload("Author").Preload("Parent").Where("id = ?", commentUUID).First(&comment); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
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

	if !idOk || !roleOk || (comment.AuthorID != authorID && roleStr != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req models.CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.Content != "" {
		comment.Content = req.Content
	}

	// Save the comment
	if result := db.DB.Save(&comment); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update comment"})
		return
	}

	// Load updated comment with associations
	db.DB.Preload("Author").Preload("Replies").Preload("Parent").Where("id = ?", commentUUID).First(&comment)

	// Clean up sensitive information
	comment.Author.Password = ""
	comment.Author.Role = ""

	c.JSON(http.StatusOK, comment)
}

// @Summary Delete comment
// @Description Delete a comment
// @Tags comments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "COMMENT ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /comment/{id} [delete]
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	id := c.Param("id")

	// Parse the UUID
	commentUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment ID format"})
		return
	}

	var comment models.Comment
	if result := db.DB.Preload("Author").Preload("Parent").Where("id = ?", commentUUID).First(&comment); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
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

	if !idOk || !roleOk || (comment.AuthorID != authorID && roleStr != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	// Delete the comment
	if result := db.DB.Delete(&comment); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment deleted successfully"})

}
